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
    private(set) var pendingCompletions: Set<String> = []

    private var pendingTasks: [String: Task<Void, Never>] = [:]
    private var completionSnapshots: [String: (chore: HearthChore, index: Int)] = [:]

    private let api: HearthAPI
    private let session: () -> String?
    let completionDelay: Duration
    private let completionCommit: @Sendable (_ token: String, _ choreID: String) async throws -> HearthChore
    private var createCommit: @Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?) async throws -> HearthChore

    init(
        api: HearthAPI = HearthAPI(),
        session: @escaping () -> String?,
        completionDelay: Duration = .seconds(3),
        completionCommit: (@Sendable (_ token: String, _ choreID: String) async throws -> HearthChore)? = nil,
        createCommit: (@Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?) async throws -> HearthChore)? = nil
    ) {
        self.api = api
        self.session = session
        self.completionDelay = completionDelay
        self.completionCommit = completionCommit ?? { token, id in
            try await api.completeChore(sessionToken: token, choreID: id)
        }
        self.createCommit = createCommit ?? { token, title, notes, claim, willCompleteBy in
            try await api.createChore(
                sessionToken: token,
                title: title,
                notes: notes,
                claim: claim,
                willCompleteBy: willCompleteBy
            )
        }
    }

    /// Optimistically remove `choreID` from the backlog, then (after
    /// `completionDelay`) hit the server. If `cancelCompletion(choreID:)` is
    /// called before the delay elapses the task is cancelled and the chore is
    /// restored at its previous index — no network call fires.
    func startCompletion(choreID: String) {
        guard pendingTasks[choreID] == nil else { return }
        guard let index = chores.firstIndex(where: { $0.id == choreID }) else { return }
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }

        let chore = chores[index]
        completionSnapshots[choreID] = (chore, index)
        chores.remove(at: index)
        pendingCompletions.insert(choreID)

        let commit = completionCommit
        let delay = completionDelay

        let task = Task { [weak self] in
            do {
                try await Task.sleep(for: delay)
                try Task.checkCancellation()
                _ = try await commit(token, choreID)
                await self?.finishCompletion(choreID: choreID)
            } catch is CancellationError {
                // Cancellation path is handled by cancelCompletion.
            } catch {
                await self?.failCompletion(choreID: choreID, error: error)
            }
        }
        pendingTasks[choreID] = task
    }

    /// Cancels a pending completion, restoring the chore at its previous index.
    func cancelCompletion(choreID: String) {
        guard let task = pendingTasks.removeValue(forKey: choreID) else { return }
        task.cancel()
        pendingCompletions.remove(choreID)
        restoreSnapshot(choreID: choreID)
    }

    private func finishCompletion(choreID: String) {
        pendingTasks[choreID] = nil
        pendingCompletions.remove(choreID)
        completionSnapshots[choreID] = nil
    }

    private func failCompletion(choreID: String, error: Error) {
        pendingTasks[choreID] = nil
        pendingCompletions.remove(choreID)
        restoreSnapshot(choreID: choreID)
        phase = .failed(error.localizedDescription)
    }

    private func restoreSnapshot(choreID: String) {
        guard let snapshot = completionSnapshots.removeValue(forKey: choreID) else { return }
        let safeIndex = min(snapshot.index, chores.count)
        chores.insert(snapshot.chore, at: safeIndex)
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

    /// Create a new chore and optimistically prepend it on success. Pass
    /// `claim: true` to atomically self-claim it in the same request; pass
    /// `willCompleteBy` (ISO-8601) alongside to stamp a commit date. The
    /// server rejects `willCompleteBy` without `claim`.
    func create(
        title: String,
        notes: String?,
        claim: Bool = false,
        willCompleteBy: String? = nil
    ) async {
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }
        isCreating = true
        defer { isCreating = false }
        do {
            let chore = try await createCommit(
                token,
                title.trimmingCharacters(in: .whitespacesAndNewlines),
                notes?.trimmingCharacters(in: .whitespacesAndNewlines).nilIfEmpty,
                claim,
                willCompleteBy
            )
            chores.insert(chore, at: 0)
            phase = .loaded
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

    /// Snooze a chore for the caller for 8 hours. Keeps the chore in the
    /// local backlog but stamps `snoozedUntil` on it so the Snoozed section
    /// grouping picks it up. The server replies 204 — we approximate the
    /// stamp locally (next reload refreshes with the real value).
    func snooze(choreID: String) async {
        guard let token = session() else { return }
        do {
            try await api.snoozeChore(sessionToken: token, choreID: choreID)
            if let idx = chores.firstIndex(where: { $0.id == choreID }) {
                chores[idx] = chores[idx].with(snoozedUntil: Self.approximateSnoozeStamp())
            }
            phase = .loaded
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    /// Wake a snoozed chore so it returns to its normal section on the next
    /// read. Idempotent on the server. Clears `snoozedUntil` locally.
    func wake(choreID: String) async {
        guard let token = session() else { return }
        do {
            try await api.wakeChore(sessionToken: token, choreID: choreID)
            if let idx = chores.firstIndex(where: { $0.id == choreID }) {
                chores[idx] = chores[idx].with(snoozedUntil: nil)
            }
            phase = .loaded
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    private static func approximateSnoozeStamp() -> String {
        let eightHours = Date().addingTimeInterval(8 * 60 * 60)
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f.string(from: eightHours)
    }

    /// Release the caller's claim on a chore. Server clears claimed_by,
    /// claimed_at, and will_complete_by. Row stays in the list, now unclaimed.
    func release(choreID: String) async {
        guard let token = session() else { return }
        do {
            let updated = try await api.releaseClaim(sessionToken: token, choreID: choreID)
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

#if DEBUG
extension BacklogStore {
    convenience init(preview chores: [HearthChore]) {
        self.init(session: { nil })
        self.chores = chores
        self.phase = .loaded
    }

    /// Preview + test-only backdoor to seed the chore list without hitting the API.
    func setChoresForTesting(_ chores: [HearthChore]) {
        self.chores = chores
        self.phase = .loaded
    }

    /// Test-only: swap the create commit closure after init so a single
    /// store can exercise both success and failure branches in one run.
    func replaceCreateCommitForTesting(
        _ commit: @escaping @Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?) async throws -> HearthChore
    ) {
        self.createCommit = commit
    }
}
#endif
