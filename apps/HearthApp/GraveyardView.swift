import SwiftUI

/// Searchable list of completed chores across the user's lists.
struct GraveyardView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: GraveyardStore

    init(initialStore: GraveyardStore? = nil) {
        _store = State(initialValue: initialStore ?? GraveyardStore(session: { nil }))
    }

    private var currentUserID: String? {
        if case .signedIn(let user) = auth.phase { return user.id }
        return nil
    }

    var body: some View {
        NavigationStack {
            content
                .navigationTitle("Done")
                .background(Color.hearthBackground)
                .scrollContentBackground(.hidden)
                .searchable(text: Binding(
                    get: { store.query },
                    set: { store.query = $0 }
                ), prompt: "Search chores")
                .onChange(of: store.query) { _, _ in
                    Task { await store.reload() }
                }
                .refreshable { await store.reload() }
                .task {
                    if case .idle = store.phase {
                        await store.reload()
                    }
                }
        }
    }

    @ViewBuilder
    private var content: some View {
        switch store.phase {
        case .idle, .loading where store.chores.isEmpty:
            VStack(spacing: 12) {
                ProgressView()
                Text("Loading…")
                    .foregroundStyle(Color.hearthTextSecondary)
                    .font(.footnote)
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

        case .loaded where store.chores.isEmpty:
            VStack(spacing: 8) {
                Image(systemName: "checkmark.circle")
                    .font(.largeTitle)
                    .foregroundStyle(Color.hearthMint)
                Text(store.query.isEmpty ? "Nothing completed yet." : "No matches.")
                    .font(.title3)
                    .foregroundStyle(Color.hearthTextPrimary)
                if store.query.isEmpty {
                    Text("Completed chores land here.")
                        .font(.footnote)
                        .foregroundStyle(Color.hearthTextSecondary)
                }
            }
            .frame(maxWidth: .infinity, maxHeight: .infinity)

        default:
            List {
                ForEach(store.chores) { chore in
                    HStack(spacing: 12) {
                        Image(systemName: "checkmark.circle.fill")
                            .foregroundStyle(Color.hearthMint)
                        VStack(alignment: .leading, spacing: 2) {
                            Text(chore.title)
                                .font(.body)
                                .foregroundStyle(Color.hearthTextPrimary)
                            if let completedAt = chore.completedAt {
                                Text(completedAt)
                                    .font(.caption)
                                    .foregroundStyle(Color.hearthTextSecondary)
                            }
                        }
                    }
                    .listRowBackground(Color.clear)
                    .listRowSeparatorTint(Color.hearthTextSecondary.opacity(0.2))
                }
            }
            .listStyle(.plain)
        }
    }
}
