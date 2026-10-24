import SwiftUI
import UIKit

struct BacklogView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: BacklogStore
    @State private var presentingCreate: Bool = false
    @State private var presentingSettings: Bool = false
    @State private var filter: Filter = .all
    @State private var dateSheetFor: HearthChore?

    init(initialStore: BacklogStore? = nil) {
        _store = State(initialValue: initialStore ?? BacklogStore(session: { nil }))
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
            .refreshable { await store.reload() }
            .task {
                if case .idle = store.phase {
                    await store.reload()
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
            List {
                ForEach(visibleChores) { chore in
                    ChoreRow(
                        chore: chore,
                        currentUserID: currentUserID,
                        onComplete: { Task { await store.complete(choreID: chore.id) } }
                    )
                    .listRowBackground(Color.clear)
                    .listRowSeparatorTint(Color.hearthTextSecondary.opacity(0.2))
                    .swipeActions(edge: .leading, allowsFullSwipe: true) {
                        if chore.claimedBy == nil {
                            Button {
                                Task {
                                    await store.claim(choreID: chore.id)
                                    // After successful claim, prompt for an
                                    // optional date commitment.
                                    if let claimed = store.chores.first(where: { $0.id == chore.id }),
                                       claimed.claimedBy == currentUserID {
                                        dateSheetFor = claimed
                                    }
                                }
                            } label: {
                                Label("Claim", systemImage: "hand.raised.fill")
                            }
                            .tint(Color.hearthLavender)
                        }
                    }
                }
            }
            .listStyle(.plain)
            .animation(.easeInOut(duration: 0.25), value: visibleChores.map(\.id))
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

#Preview("Empty") {
    let store = BacklogStore(session: { "token" })
    return BacklogView(initialStore: store)
        .environment(AuthStore())
}
