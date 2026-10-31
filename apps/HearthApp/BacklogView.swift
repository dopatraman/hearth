import SwiftUI
import UIKit

struct BacklogView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: BacklogStore
    private let members: MembersStore
    @State private var isComposing: Bool = false
    @State private var presentingSettings: Bool = false
    @State private var filter: BacklogFilter = .all
    @State private var dateSheetFor: HearthChore?
    @State private var didLoad: Bool = false
    @State private var openID: String?
    @State private var toast: ToastModel?
    @State private var completingID: String?

    init(initialStore: BacklogStore? = nil, members: MembersStore? = nil) {
        _store = State(initialValue: initialStore ?? BacklogStore(session: { nil }))
        self.members = members ?? MembersStore(session: { nil })
    }

    private var currentUserID: String? {
        if case .signedIn(let user) = auth.phase { return user.id }
        return nil
    }

    private var visibleChores: [HearthChore] {
        switch filter {
        case .all:
            return store.chores
        case .member(let id):
            return store.chores.filter { $0.claimedBy == id }
        }
    }

    private var sections: [BacklogSection] {
        groupedSections(chores: store.chores, currentUserID: currentUserID, filter: filter)
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                BacklogHeader(
                    chores: store.chores,
                    members: members.members,
                    currentUserID: currentUserID,
                    onSettings: { presentingSettings = true },
                    selectedMemberID: filter.selectedMemberID,
                    onTapMember: { member in
                        withAnimation(.spring(response: 0.4, dampingFraction: 0.82)) {
                            if filter.selectedMemberID == member.id {
                                filter = .all
                            } else {
                                filter = .member(member.id)
                            }
                        }
                    }
                )
                .padding(.bottom, 10)

                content
            }
            .scaleEffect(isComposing ? 0.96 : 1)
            .blur(radius: isComposing ? 2 : 0)
            .animation(.timingCurve(0.2, 0.8, 0.2, 1, duration: 0.5), value: isComposing)
            .toolbar(.hidden, for: .navigationBar)
            .background(Color.hearthBackground)
            .scrollContentBackground(.hidden)
            .refreshable {
                await Task {
                    await store.reload()
                    await members.load()
                }.value
            }
            .task {
                if case .idle = store.phase {
                    await store.reload()
                }
                if members.members.isEmpty {
                    await members.load()
                }
                withAnimation { didLoad = true }
            }
            .sheet(isPresented: $presentingSettings) {
                SettingsSheet()
            }
            .sheet(item: $dateSheetFor) { chore in
                ClaimWhenSheet(choreTitle: chore.title) { date in
                    if let date {
                        await store.setWillCompleteBy(choreID: chore.id, when: date)
                    }
                }
            }
            .overlay {
                if isComposing {
                    Color.black.opacity(0.22)
                        .ignoresSafeArea()
                        .onTapGesture { isComposing = false }
                        .transition(.opacity)
                }
            }
            .animation(.easeInOut(duration: 0.4), value: isComposing)
            .overlay(alignment: .bottom) {
                NewChoreCard(isOpen: $isComposing) { title, notes, claim, bucket in
                    let iso = bucket?.resolve(now: Date()).map { WhenishBucket.iso8601(from: $0) }
                    await store.create(title: title, notes: notes, claim: claim, willCompleteBy: iso)
                    showCreationToast(claimed: claim, bucket: bucket)
                }
            }
            .overlay(alignment: .bottom) {
                if let toast {
                    ToastView(
                        toast: toast,
                        onUndo: toast.undoLabel != nil ? { undoLastCompletion() } : nil
                    )
                    .padding(.horizontal, 20)
                    .padding(.bottom, 90)
                    .transition(.move(edge: .bottom).combined(with: .opacity))
                    .task(id: toast.id) {
                        try? await Task.sleep(for: .milliseconds(3200))
                        if self.toast?.id == toast.id {
                            withAnimation(.easeOut(duration: 0.3)) {
                                self.toast = nil
                                self.lastCompletedChoreID = nil
                            }
                        }
                    }
                }
            }
        }
    }

    @State private var lastCompletedChoreID: String?

    private func undoLastCompletion() {
        guard let id = lastCompletedChoreID else { return }
        store.cancelCompletion(choreID: id)
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        withAnimation(.easeOut(duration: 0.3)) {
            toast = nil
            lastCompletedChoreID = nil
        }
    }

    private func beginCompletion(choreID: String) {
        guard completingID == nil else { return }
        UINotificationFeedbackGenerator().notificationOccurred(.success)
        lastCompletedChoreID = choreID
        openID = nil
        withAnimation(.spring(response: 0.3, dampingFraction: 0.72)) {
            completingID = choreID
        }
        Task {
            try? await Task.sleep(for: .milliseconds(450))
            withAnimation(.spring(response: 0.45, dampingFraction: 0.82)) {
                completingID = nil
                store.startCompletion(choreID: choreID)
                toast = ToastModel(text: "Moved to Done", undoLabel: "Undo")
            }
        }
    }

    private func showClaimToast() {
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        withAnimation(.spring(response: 0.4, dampingFraction: 0.78)) {
            toast = ToastModel(text: "Yours.", undoLabel: nil)
        }
    }

    private func showCreationToast(claimed: Bool, bucket: WhenishBucket?) {
        if case .failed = store.phase { return }
        let text: String
        if claimed {
            text = "Added. Yours" + (bucket?.toastSuffix ?? "")
        } else {
            text = "Added. Up for grabs"
        }
        withAnimation(.spring(response: 0.4, dampingFraction: 0.78)) {
            toast = ToastModel(text: text, kind: .creation)
        }
    }

    @ViewBuilder
    private var content: some View {
        switch store.phase {
        case .idle, .loading where store.chores.isEmpty:
            VStack(spacing: 12) {
                ProgressView()
                Text("Loading backlog…")
                    .foregroundStyle(Color.hearthTextSecondary)
                    .font(.footnote)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

        case .loaded where visibleChores.isEmpty:
            emptyState

        case .failed(let message) where store.chores.isEmpty:
            VStack(spacing: 12) {
                Image(systemName: "exclamationmark.triangle")
                    .font(.largeTitle)
                    .foregroundStyle(Color.hearthTextSecondary)
                Text(message)
                    .font(.footnote)
                    .foregroundStyle(Color.hearthTextSecondary)
                    .multilineTextAlignment(.center)
                    .padding(.horizontal, 32)
                Button("Try again") { Task { await store.reload() } }
                    .buttonStyle(.borderedProminent)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

        default:
            loadedList
        }
    }

    private var loadedList: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 18) {
                ForEach(Array(sections.enumerated()), id: \.element.id) { index, section in
                    sectionView(section, index: index)
                }
            }
            .padding(.horizontal, 16)
            .padding(.top, 6)
            .padding(.bottom, 150)
        }
        .animation(.spring(response: 0.45, dampingFraction: 0.82), value: sections)
    }

    private func sectionView(_ section: BacklogSection, index: Int) -> some View {
        VStack(alignment: .leading, spacing: 0) {
            SectionHeader(
                label: section.label,
                dotColor: section.dotColor,
                count: section.items.count
            )
            VStack(spacing: 10) {
                ForEach(section.items) { chore in
                    choreRow(for: chore)
                        .transition(
                            .asymmetric(
                                insertion: .scale(scale: 0.94).combined(with: .opacity),
                                removal: .opacity.combined(with: .offset(x: 28))
                            )
                        )
                }
            }
        }
        .opacity(didLoad ? 1 : 0)
        .offset(y: didLoad ? 0 : 10)
        .animation(
            .easeOut(duration: 0.55).delay(Double(index) * 0.06),
            value: didLoad
        )
    }

    private func choreRow(for chore: HearthChore) -> some View {
        ChoreRow(
            chore: chore,
            currentUserID: currentUserID,
            members: members.members,
            isOpen: openID == chore.id,
            onToggle: {
                withAnimation(.spring(response: 0.4, dampingFraction: 0.82)) {
                    openID = openID == chore.id ? nil : chore.id
                }
            },
            onComplete: {
                beginCompletion(choreID: chore.id)
            },
            onClaim: {
                Task {
                    await store.claim(choreID: chore.id)
                    if let claimed = store.chores.first(where: { $0.id == chore.id }),
                       claimed.claimedBy == currentUserID {
                        openID = nil
                        dateSheetFor = claimed
                        showClaimToast()
                    }
                }
            },
            onSetDate: {
                openID = nil
                dateSheetFor = chore
            },
            onSnooze: {
                UIImpactFeedbackGenerator(style: .soft).impactOccurred()
                withAnimation(.spring(response: 0.4, dampingFraction: 0.82)) {
                    openID = nil
                }
                Task { await store.snooze(choreID: chore.id) }
            },
            onRelease: {
                UIImpactFeedbackGenerator(style: .soft).impactOccurred()
                openID = nil
                Task { await store.release(choreID: chore.id) }
            },
            onWake: {
                UIImpactFeedbackGenerator(style: .soft).impactOccurred()
                withAnimation(.spring(response: 0.4, dampingFraction: 0.82)) {
                    openID = nil
                }
                Task { await store.wake(choreID: chore.id) }
            },
            isCompleting: completingID == chore.id
        )
    }

    private var emptyState: some View {
        let filteringToMe = filter.selectedMemberID == currentUserID && currentUserID != nil
        let filteringToOther = filter.selectedMemberID != nil && !filteringToMe
        let title: String = {
            if filteringToMe { return "No chores claimed." }
            if filteringToOther { return "Nothing here." }
            return "No chores yet."
        }()
        let subtitle: String = {
            if filteringToMe { return "Swipe right on a chore to claim it." }
            if filteringToOther { return "Tap again to clear the filter." }
            return "Tap + to add one."
        }()
        return VStack(spacing: 8) {
            Text(title)
                .font(.title3)
                .foregroundStyle(Color.hearthTextPrimary)
            Text(subtitle)
                .font(.footnote)
                .foregroundStyle(Color.hearthTextSecondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

#if DEBUG
private func previewChore(
    id: String,
    title: String,
    notes: String? = nil,
    claimedBy: String? = nil,
    willCompleteBy: String? = nil,
    overdue: Bool = false,
    createdAt: String = "2026-10-20T10:00:00.000Z"
) -> HearthChore {
    HearthChore(
        id: id,
        listId: "list-preview",
        creatorId: "u_self",
        title: title,
        notes: notes,
        state: "open",
        claimedBy: claimedBy,
        claimedAt: claimedBy == nil ? nil : "2026-10-23T14:00:00.000Z",
        willCompleteBy: willCompleteBy,
        completedBy: nil,
        completedAt: nil,
        createdAt: createdAt,
        updatedAt: createdAt,
        overdue: overdue
    )
}

#Preview("Populated — All") {
    let selfUser = HearthUser(id: "u_self", displayName: "Venkat", notificationsEnabled: true)
    let members = [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]
    let chores: [HearthChore] = [
        previewChore(id: "c1", title: "Clean out the fridge",
                     notes: "Toss anything past its date. Wipe the shelves.",
                     claimedBy: "u_self",
                     willCompleteBy: "2026-10-23T23:59:59.000Z",
                     overdue: true),
        previewChore(id: "c2", title: "Mop the kitchen",
                     claimedBy: "u_maya",
                     willCompleteBy: "2026-10-24T23:59:59.000Z",
                     overdue: true),
        previewChore(id: "c3", title: "Take out the bins",
                     notes: "Recycling and compost. Pickup is 7am.",
                     claimedBy: "u_self",
                     willCompleteBy: "2026-10-25T23:59:59.000Z"),
        previewChore(id: "c4", title: "Water the plants", claimedBy: "u_self"),
        previewChore(id: "c5", title: "Wipe down the stove"),
        previewChore(id: "c6", title: "Change the bedsheets", notes: "Guest room too."),
        previewChore(id: "c7", title: "Order more dish soap"),
        previewChore(id: "c8", title: "Vacuum the stairs",
                     claimedBy: "u_maya",
                     willCompleteBy: "2026-10-26T23:59:59.000Z"),
    ]
    let auth = AuthStore(previewSignedInUser: selfUser)
    let backlog = BacklogStore(preview: chores)
    let memberStore = MembersStore(preview: members)
    return BacklogView(initialStore: backlog, members: memberStore)
        .environment(auth)
}

#Preview("Populated — Solo household") {
    let selfUser = HearthUser(id: "u_self", displayName: "Venkat", notificationsEnabled: true)
    let members = [HearthMember(id: "u_self", displayName: "Venkat")]
    let chores: [HearthChore] = [
        previewChore(id: "s1", title: "Buy coffee"),
        previewChore(id: "s2", title: "Take out trash", claimedBy: "u_self",
                     willCompleteBy: "2026-10-25T23:59:59.000Z"),
    ]
    let auth = AuthStore(previewSignedInUser: selfUser)
    let backlog = BacklogStore(preview: chores)
    let memberStore = MembersStore(preview: members)
    return BacklogView(initialStore: backlog, members: memberStore)
        .environment(auth)
}

#Preview("Empty") {
    let auth = AuthStore(previewSignedInUser:
        HearthUser(id: "u_self", displayName: "Venkat", notificationsEnabled: true))
    let backlog = BacklogStore(preview: [])
    let memberStore = MembersStore(preview: [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ])
    return BacklogView(initialStore: backlog, members: memberStore)
        .environment(auth)
}
#endif
