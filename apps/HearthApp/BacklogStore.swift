import Foundation

/// BacklogStore owns the current list of chores and the loading / mutation
/// operations against the server. SwiftUI views observe this; it owns no UI.
@Observable
@MainActor
final class BacklogStore {
    enum Phase: Equatable {
        case idle
        case loading
        case loaded
        case failed(String)
    }

    private(set) var chores: [HearthChore] = []
    private(set) var phase: Phase = .idle
    private(set) var isCreating: Bool = false

    private let api: HearthAPI
    private let session: () -> String?

    init(api: HearthAPI = HearthAPI(), session: @escaping () -> String?) {
        self.api = api
        self.session = session
    }

    /// Load (or reload) the backlog. Safe to call repeatedly — overwrites
    /// chores on success, leaves them unchanged on failure.
    func reload() async {
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }
        phase = .loading
        do {
            chores = try await api.backlog(sessionToken: token)
            phase = .loaded
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    /// Create a new chore and optimistically prepend it on success. On
    /// failure, `phase` surfaces the error without touching the list.
    func create(title: String, notes: String?) async {
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }
        isCreating = true
        defer { isCreating = false }

        do {
            let chore = try await api.createChore(
                sessionToken: token,
                title: title.trimmingCharacters(in: .whitespacesAndNewlines),
                notes: notes?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty
            )
            chores.insert(chore, at: 0)
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }
}

private extension String {
    var nilIfEmpty: String? { isEmpty ? nil : self }
}
