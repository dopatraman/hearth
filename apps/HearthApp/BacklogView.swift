import SwiftUI
import UIKit

struct BacklogView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: BacklogStore
    private let members: MembersStore
    @State private var presentingCreate: Bool = false
    @State private var presentingSettings: Bool = false
    @State private var filter: Filter = .all
    @State private var dateSheetFor: HearthChore?

    init(initialStore: BacklogStore? = nil, members: MembersStore? = nil) {
        _store = State(initialValue: initialStore ?? BacklogStore(session: { nil }))
        self.members = members ?? MembersStore(session: { nil })
    }

    enum Filter: Hashable {
        case all
        case mine
    }

    private var currentUserID: String? {
        if case .signedIn(let user) = auth.phase { return user.id }
        return nil
    }

    private var visibleChores: [HearthChore] {
        switch filter {
        case .all:
            return store.chores
        case .mine:
            guard let me = currentUserID else { return [] }
            return store.chores.filter { $0.claimedBy == me }
        }
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 0) {
                Picker("Filter", selection: $filter) {
                    Text("All").tag(Filter.all)
                    Text("Mine").tag(Filter.mine)
                }
                .pickerStyle(.segmented)
                .padding(.horizontal, 16)
                .padding(.vertical, 8)
                .sensoryFeedback(.selection, trigger: filter)

                content
            }
            .navigationTitle("Backlog")
            .toolbar {
                ToolbarItem(placement: .topBarLeading) {
                    AvatarStack(members: members.members, currentUserID: currentUserID)
                }
                ToolbarItem(placement: .topBarTrailing) {
                    Button {
                        presentingSettings = true
                    } label: {
                        Image(systemName: "gearshape")
                    }
                }
            }
            .background(Color.hearthBackground)
            .scrollContentBackground(.hidden)
            .refreshable {
                await store.reload()
                await members.load()
            }
            .task {
                if case .idle = store.phase {
                    await store.reload()
                }
                if members.members.isEmpty {
                    await members.load()
                }
            }
            .sheet(isPresented: $presentingCreate) {
                CreateChoreSheet(onSave: { title, notes in
                    await store.create(title: title, notes: notes)
                })
            }
            .sheet(isPresented: $presentingSettings) {
                SettingsSheet()
            }
            .sheet(item: $dateSheetFor) { chore in
                ClaimDateSheet(choreTitle: chore.title) { date in
                    if let date {
                        await store.setWillCompleteBy(choreID: chore.id, when: date)
                    }
                }
            }
            .overlay(alignment: .bottomTrailing) {
                FABButton {
                    UIImpactFeedbackGenerator(style: .light).impactOccurred()
                    presentingCreate = true
                }
                .padding(24)
            }
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
        List {
            ForEach(visibleChores) { chore in
                choreRow(for: chore)
            }
        }
        .listStyle(.plain)
        .animation(.easeInOut(duration: 0.25), value: visibleChores.map(\.id))
    }

    private func choreRow(for chore: HearthChore) -> some View {
        ChoreRow(
            chore: chore,
            currentUserID: currentUserID,
            members: members.members,
            onComplete: { Task { await store.complete(choreID: chore.id) } }
        )
        .listRowBackground(Color.clear)
        .listRowInsets(EdgeInsets())
        .listRowSeparatorTint(Color.hearthTextSecondary.opacity(0.2))
        .swipeActions(edge: .trailing, allowsFullSwipe: true) {
            trailingSwipeActions(for: chore)
        }
        .swipeActions(edge: .leading, allowsFullSwipe: true) {
            leadingSwipeActions(for: chore)
        }
    }

    @ViewBuilder
    private func trailingSwipeActions(for chore: HearthChore) -> some View {
        if filter == .mine {
            Button {
                UIImpactFeedbackGenerator(style: .soft).impactOccurred()
                Task { await store.snooze(choreID: chore.id) }
            } label: {
                Label("Snooze", systemImage: "moon.zzz.fill")
            }
            .tint(Color.hearthSky)
        }
    }

    @ViewBuilder
    private func leadingSwipeActions(for chore: HearthChore) -> some View {
        if chore.claimedBy == nil {
            Button {
                Task {
                    await store.claim(choreID: chore.id)
                    if let claimed = store.chores.first(where: { $0.id == chore.id }),
                       claimed.claimedBy == currentUserID {
                        dateSheetFor = claimed
                    }
                }
            } label: {
                Label("Claim", systemImage: "hand.raised.fill")
            }
            .tint(Color.hearthLavender)
        } else if chore.claimedBy == currentUserID {
            Button {
                UIImpactFeedbackGenerator(style: .soft).impactOccurred()
                Task { await store.release(choreID: chore.id) }
            } label: {
                Label("Release", systemImage: "hand.raised.slash.fill")
            }
            .tint(Color.hearthPeach)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Text(filter == .mine ? "No chores claimed." : "No chores yet.")
                .font(.title3)
                .foregroundStyle(Color.hearthTextPrimary)
            Text(filter == .mine ? "Swipe right on a chore to claim it." : "Tap + to add one.")
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
