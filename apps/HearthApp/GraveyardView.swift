import SwiftUI

/// Done tab: search + day-grouped completed chores.
struct GraveyardView: View {
    @Environment(AuthStore.self) private var auth
    @State private var store: GraveyardStore
    private let members: MembersStore

    init(initialStore: GraveyardStore? = nil, members: MembersStore? = nil) {
        _store = State(initialValue: initialStore ?? GraveyardStore(session: { nil }))
        self.members = members ?? MembersStore(session: { nil })
    }

    private var currentUserID: String? {
        if case .signedIn(let user) = auth.phase { return user.id }
        return nil
    }

    private var sections: [GraveyardGroup] {
        groupedGraveyard(chores: store.chores, now: Date())
    }

    var body: some View {
        NavigationStack {
            VStack(spacing: 18) {
                header
                searchField
                content
            }
            .padding(.horizontal, 20)
            .padding(.top, 8)
            .frame(maxWidth: .infinity, maxHeight: .infinity, alignment: .top)
            .background(Color.hearthBackground)
            .toolbar(.hidden, for: .navigationBar)
            .onChange(of: store.query) { _, _ in
                Task { await store.reload() }
            }
            .refreshable { await store.reload() }
            .task {
                if case .idle = store.phase {
                    await store.reload()
                }
                if members.members.isEmpty {
                    await members.load()
                }
            }
        }
    }

    private var header: some View {
        VStack(alignment: .leading, spacing: 0) {
            Text("Done")
                .font(.hearthDisplay(size: 36, weight: .bold))
                .foregroundStyle(Color.hearthTextPrimary)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
    }

    private var searchField: some View {
        HStack(spacing: 10) {
            Image(systemName: "magnifyingglass")
                .font(.system(size: 15, weight: .medium))
                .foregroundStyle(Color.hearthTextSecondary)
            TextField(
                "Search everything you've done",
                text: Binding(
                    get: { store.query },
                    set: { store.query = $0 }
                )
            )
            .textFieldStyle(.plain)
            .font(.hearthBody(size: 15))
            .foregroundStyle(Color.hearthTextPrimary)
            .submitLabel(.search)
        }
        .padding(.horizontal, 16)
        .frame(height: 46)
        .background(Color.hearthInsetFill, in: RoundedRectangle(cornerRadius: 16, style: .continuous))
    }

    @ViewBuilder
    private var content: some View {
        switch store.phase {
        case .idle, .loading where store.chores.isEmpty:
            loadingState

        case .loaded where store.chores.isEmpty:
            emptyState

        case .failed(let message) where store.chores.isEmpty:
            errorState(message)

        default:
            groupedList
        }
    }

    private var loadingState: some View {
        VStack(spacing: 12) {
            ProgressView()
            Text("Loading…")
                .foregroundStyle(Color.hearthTextSecondary)
                .font(.footnote)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var emptyState: some View {
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
    }

    private func errorState(_ message: String) -> some View {
        VStack(spacing: 12) {
            Image(systemName: "exclamationmark.triangle")
                .font(.largeTitle)
                .foregroundStyle(Color.hearthTextSecondary)
            Text(message)
                .font(.footnote)
                .foregroundStyle(Color.hearthTextSecondary)
                .multilineTextAlignment(.center)
            Button("Try again") { Task { await store.reload() } }
                .buttonStyle(.borderedProminent)
        }
        .frame(maxWidth: .infinity, maxHeight: .infinity)
    }

    private var groupedList: some View {
        ScrollView {
            LazyVStack(alignment: .leading, spacing: 22) {
                ForEach(sections) { section in
                    sectionView(section)
                }
            }
            .padding(.bottom, 100)
        }
        .scrollDismissesKeyboard(.interactively)
    }

    private func sectionView(_ section: GraveyardGroup) -> some View {
        VStack(alignment: .leading, spacing: 6) {
            Text(section.label)
                .font(.hearthBody(size: 14, weight: .semibold))
                .foregroundStyle(Color.hearthTextPrimary)
                .padding(.leading, 4)
            VStack(spacing: 0) {
                ForEach(Array(section.items.enumerated()), id: \.element.id) { index, chore in
                    GraveyardRow(
                        chore: chore,
                        currentUserID: currentUserID,
                        members: members.members,
                        showsBottomDivider: index < section.items.count - 1
                    )
                }
            }
            .padding(.horizontal, 16)
            .background(
                RoundedRectangle(cornerRadius: 22, style: .continuous)
                    .fill(Color.hearthSurface)
                    .shadow(color: .black.opacity(0.04), radius: 2, y: 1)
                    .shadow(color: .black.opacity(0.04), radius: 18, y: 6)
            )
        }
    }
}
