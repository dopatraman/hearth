import SwiftUI

struct BacklogView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: BacklogStore
    @State private var presentingCreate: Bool = false

    init(initialStore: BacklogStore? = nil) {
        _store = State(initialValue: initialStore ?? BacklogStore(session: { nil }))
    }

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Backlog")
                .toolbar {
                    ToolbarItem(placement: .topBarTrailing) {
                        Button {
                            auth.signOut()
                        } label: {
                            Image(systemName: "person.crop.circle")
                        }
                    }
                }
                .background(Color.hearthBackground)
                .scrollContentBackground(.hidden)
                .refreshable { await store.reload() }
                .task {
                    // First-appear load. BacklogStore uses the current session
                    // token lazily; wiring is replaced by rootStore(for:) when
                    // used from ContentView.
                    if case .idle = store.phase {
                        await store.reload()
                    }
                }
                .sheet(isPresented: $presentingCreate) {
                    CreateItemSheet(onSave: { title, notes in
                        await store.create(title: title, notes: notes)
                    })
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
        case .idle, .loading where store.items.isEmpty:
            VStack(spacing: 12) {
                ProgressView()
                Text("Loading backlog…")
                    .foregroundStyle(Color.hearthTextSecondary)
                    .font(.footnote)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

        case .loaded where store.items.isEmpty:
            emptyState

        case .failed(let message) where store.items.isEmpty:
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
                ForEach(store.items) { item in
                    ItemRow(item: item)
                        .listRowBackground(Color.clear)
                        .listRowSeparatorTint(Color.hearthTextSecondary.opacity(0.2))
                }
            }
            .listStyle(.plain)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 8) {
            Text("No chores yet.")
                .font(.title3)
                .foregroundStyle(Color.hearthTextPrimary)
            Text("Tap + to add one.")
                .font(.footnote)
                .foregroundStyle(Color.hearthTextSecondary)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }
}

// MARK: - Row

private struct ItemRow: View {
    let item: HearthItem

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(item.title)
                .font(.body)
                .foregroundStyle(Color.hearthTextPrimary)
            if let notes = item.notes, !notes.isEmpty {
                Text(notes)
                    .font(.footnote)
                    .foregroundStyle(Color.hearthTextSecondary)
                    .lineLimit(2)
            }
        }
        .padding(.vertical, 6)
    }
}

// MARK: - FAB

private struct FABButton: View {
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Image(systemName: "plus")
                .font(.system(size: 24, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
                .frame(width: 60, height: 60)
                .background(.ultraThinMaterial, in: Circle())
                .overlay(Circle().stroke(Color.hearthTextSecondary.opacity(0.2), lineWidth: 0.5))
                .shadow(color: Color.black.opacity(0.1), radius: 10, y: 4)
        }
        .buttonStyle(.plain)
        .sensoryFeedback(.impact(weight: .light), trigger: UUID())
    }
}

#Preview("Empty") {
    let store = BacklogStore(session: { "token" })
    return BacklogView(initialStore: store)
        .environment(AuthStore())
}
