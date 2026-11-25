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
    /// Ids of recurring chores the server just replenished — the rail watches
    /// this set to play the "back" animation on the returning tile. Cleared
    /// shortly after the row is reinserted, or on any reload.
    private(set) var recentlyReplenished: Set<String> = []

    private var pendingTasks: [String: Task<Void, Never>] = [:]
    private var completionSnapshots: [String: (chore: HearthChore, index: Int)] = [:]
    private var replenishClearTasks: [String: Task<Void, Never>] = [:]
    private var reorderLifted: Bool = false
    private var reloadDeferred: Bool = false

    private let api: HearthAPI
    private let session: () -> String?
    let completionDelay: Duration
    private let replenishFlashDuration: Duration
    private let completionCommit: @Sendable (_ token: String, _ choreID: String) async throws -> HearthChore
    private var createCommit: @Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?, _ isRecurring: Bool, _ icon: String?) async throws -> HearthChore

    init(
        api: HearthAPI = HearthAPI(),
        session: @escaping () -> String?,
        completionDelay: Duration = .seconds(3),
        replenishFlashDuration: Duration = .milliseconds(1200),
        completionCommit: (@Sendable (_ token: String, _ choreID: String) async throws -> HearthChore)? = nil,
        createCommit: (@Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?, _ isRecurring: Bool, _ icon: String?) async throws -> HearthChore)? = nil
    ) {
        self.api = api
        self.session = session
        self.completionDelay = completionDelay
        self.replenishFlashDuration = replenishFlashDuration
        self.completionCommit = completionCommit ?? { token, id in
            try await api.completeChore(sessionToken: token, choreID: id)
        }
        self.createCommit = createCommit ?? { token, title, notes, claim, willCompleteBy, isRecurring, icon in
            try await api.createChore(
                sessionToken: token,
                title: title,
                notes: notes,
                claim: claim,
                willCompleteBy: willCompleteBy,
                isRecurring: isRecurring,
                icon: icon
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
                let finalized = try await commit(token, choreID)
                await self?.finishCompletion(choreID: choreID, finalized: finalized)
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

    private func finishCompletion(choreID: String, finalized: HearthChore) {
        pendingTasks[choreID] = nil
        pendingCompletions.remove(choreID)
        let snapshot = completionSnapshots.removeValue(forKey: choreID)
        // Recurring chores come back from the server as the same row with the
        // claim cleared (state still 'open'). Reinsert so the Backlog's
        // Recurring rail sees the tile return.
        if finalized.state == "open" && finalized.isRecurring {
            if !chores.contains(where: { $0.id == finalized.id }) {
                let insertion = snapshot?.index ?? 0
                let safeIndex = min(insertion, chores.count)
                chores.insert(finalized, at: safeIndex)
            }
            flagReplenished(id: finalized.id)
        }
    }

    private func flagReplenished(id: String) {
        recentlyReplenished.insert(id)
        replenishClearTasks[id]?.cancel()
        let duration = replenishFlashDuration
        let task = Task { [weak self] in
            try? await Task.sleep(for: duration)
            await self?.clearReplenished(id: id)
        }
        replenishClearTasks[id] = task
    }

    private func clearReplenished(id: String) {
        replenishClearTasks[id] = nil
        recentlyReplenished.remove(id)
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
    /// unchanged on failure. Suspended while a reorder drag is in flight so
    /// the user's drag state doesn't desync with a mid-drag refresh; any
    /// deferred reload fires once the drag ends.
    func reload() async {
        if reorderLifted {
            reloadDeferred = true
            return
        }
        guard let token = session() else {
            phase = .failed("Not signed in")
            return
        }
        if chores.isEmpty { phase = .loading }
        do {
            chores = try await api.backlog(sessionToken: token)
            phase = .loaded
        } catch {
            phase = .failed(error.localizedDescription)
        }
    }

    /// Called by the Backlog view when a reorder lift starts/ends. Suspends
    /// refreshes while the drag is live so the view's local drag indices stay
    /// aligned with the chore list it rendered against.
    func setReorderLifted(_ lifted: Bool) {
        reorderLifted = lifted
        if !lifted && reloadDeferred {
            reloadDeferred = false
            Task { await reload() }
        }
    }

    /// Create a new chore and optimistically prepend it on success. Pass
    /// `claim: true` to atomically self-claim it in the same request; pass
    /// `willCompleteBy` (ISO-8601) alongside to stamp a commit date. The
    /// server rejects `willCompleteBy` without `claim`. Pass `isRecurring:
    /// true` for a chore that lives in the rail — the server rejects a
    /// recurring request that also tries to claim.
    func create(
        title: String,
        notes: String?,
        claim: Bool = false,
        willCompleteBy: String? = nil,
        isRecurring: Bool = false,
        icon: String? = nil
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
                willCompleteBy,
                isRecurring,
                icon
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

    /// Rewrite one section's order globally. The server infers which section
    /// from the ids (all must be in the same one). Snapshots the current
    /// chores for rollback, bumps every submitted id's `sectionOrder` to its
    /// new 1..N position locally so the UI stays put while the request is in
    /// flight, then POSTs. On success the response replaces local state. On
    /// failure the snapshot is restored.
    @discardableResult
    func reorderSection(_ order: [String]) async -> Bool {
        guard let token = session() else {
            phase = .failed("Not signed in")
            return false
        }
        let snapshot = chores
        let ranks = Dictionary(uniqueKeysWithValues: order.enumerated().map { ($0.element, $0.offset + 1) })
        chores = chores.map { chore in
            guard let newRank = ranks[chore.id] else { return chore }
            return chore.with(sectionOrder: newRank)
        }
        do {
            let updated = try await api.reorderChores(sessionToken: token, order: order)
            chores = updated
            return true
        } catch {
            chores = snapshot
            phase = .failed(error.localizedDescription)
            return false
        }
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
        _ commit: @escaping @Sendable (_ token: String, _ title: String, _ notes: String?, _ claim: Bool, _ willCompleteBy: String?, _ isRecurring: Bool, _ icon: String?) async throws -> HearthChore
    ) {
        self.createCommit = commit
    }
}
#endif
