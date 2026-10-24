import Foundation

/// BacklogStore owns the current list of open chores and the mutation
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

    /// Load (or reload) the backlog. Overwrites chores on success, leaves them
    /// unchanged on failure.
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

    /// Create a new chore and optimistically prepend it on success.
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

    /// Claim a chore. On success, the chore stays in the list but with
    /// claimed_by set — row visuals update.
    func claim(choreID: String) async {
        guard let token = session() else { return }
        do {
            let updated = try await api.claimChore(sessionToken: token, choreID: choreID)
            if let idx = chores.firstIndex(where: { $0.id == choreID }) {
                chores[idx] = updated
            }
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    /// Complete a chore. On success, removes it from the backlog (only open
    /// chores live here) and returns the completed chore for post-removal
    /// animations.
    /// Set (or clear with nil) the will_complete_by for a chore. Must already
    /// be claimed by the caller server-side.
    func setWillCompleteBy(choreID: String, when: Date?) async {
        guard let token = session() else { return }
        do {
            let updated = try await api.setWillCompleteBy(sessionToken: token, choreID: choreID, when: when)
            if let idx = chores.firstIndex(where: { $0.id == choreID }) {
                chores[idx] = updated
            }
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    @discardableResult
    func complete(choreID: String) async -> HearthChore? {
        guard let token = session() else { return nil }
        do {
            let completed = try await api.completeChore(sessionToken: token, choreID: choreID)
            chores.removeAll { $0.id == choreID }
            return completed
        } catch {
            phase = .failed(error.localizedDescription)
            return nil
        }
    }
}

private extension String {
    var nilIfEmpty: String? { isEmpty ? nil : self }
}
