import Foundation

/// GraveyardStore owns the completed-chores list for the Graveyard tab.
/// Supports a search query. Reloads lazily on tab-focus or query-change.
@Observable
@MainActor
final class GraveyardStore {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case failed(String)
    }

    private(set) var chores: [HearthChore] = []
    private(set) var phase: Phase = .idle
    var query: String = ""

    private let api: HearthAPI
    private let session: () -> String?

    init(api: HearthAPI = HearthAPI(), session: @escaping () -> String?) {
        self.api = api
        self.session = session
    }

    /// Load (or reload). Uses the current query.
    func reload() async {
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }
        if chores.isEmpty { phase = .loading }
        do {
            let q = query.trimmingCharacters(in: .whitespacesAndNewlines)
            chores = try await api.graveyard(sessionToken: token, query: q.isEmpty ? nil : q)
            phase = .loaded
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }
}
