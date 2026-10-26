import Foundation
import Testing
@testable import HearthApp

@MainActor
@Suite("BacklogStore completion queue")
struct BacklogStoreCompletionTests {

    private func makeStore(
        chores: [HearthChore],
        delay: Duration = .milliseconds(10),
        recorder: CallRecorder = CallRecorder()
    ) -> BacklogStore {
        let store = BacklogStore(
            session: { "token" },
            completionDelay: delay,
            completionCommit: { _, id in
                try await recorder.record(id)
                return chores.first(where: { $0.id == id })!
            }
        )
        store.setChoresForTesting(chores)
        return store
    }

    private func chore(_ id: String) -> HearthChore {
        HearthChore(
            id: id, listId: "l", creatorId: "me", title: id,
            notes: nil, state: "open",
            claimedBy: "me", claimedAt: nil, willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "", updatedAt: "", overdue: false
        )
    }

    @Test("startCompletion removes chore locally immediately")
    func startCompletion_removesChoreLocallyImmediately() async {
        let store = makeStore(chores: [chore("c0"), chore("c1"), chore("c2")])
        store.startCompletion(choreID: "c1")

        #expect(store.chores.map(\.id) == ["c0", "c2"])
        #expect(store.pendingCompletions.contains("c1"))
    }

    @Test("cancelCompletion restores chore at original index")
    func cancelCompletion_restoresChoreAtOriginalIndex() async {
        let store = makeStore(chores: [chore("c0"), chore("c1"), chore("c2")],
                              delay: .seconds(5))
        store.startCompletion(choreID: "c1")
        store.cancelCompletion(choreID: "c1")

        #expect(store.chores.map(\.id) == ["c0", "c1", "c2"])
        #expect(store.pendingCompletions.contains("c1") == false)
    }

    @Test("startCompletion fires API after delay")
    func startCompletion_firesAPIAfterDelay() async throws {
        let recorder = CallRecorder()
        let store = makeStore(chores: [chore("c1")],
                              delay: .milliseconds(10),
                              recorder: recorder)

        store.startCompletion(choreID: "c1")
        try await Task.sleep(for: .milliseconds(80))

        let calls = await recorder.calls
        #expect(calls == ["c1"])
        #expect(store.pendingCompletions.contains("c1") == false)
    }

    @Test("startCompletion does not fire API if cancelled before delay")
    func startCompletion_doesNotFireAPIifCancelledBeforeDelay() async throws {
        let recorder = CallRecorder()
        let store = makeStore(chores: [chore("c1")],
                              delay: .seconds(1),
                              recorder: recorder)

        store.startCompletion(choreID: "c1")
        store.cancelCompletion(choreID: "c1")
        try await Task.sleep(for: .milliseconds(100))

        let calls = await recorder.calls
        #expect(calls.isEmpty)
    }

    @Test("startCompletion called twice for same chore is idempotent")
    func startCompletion_calledTwiceForSameChore_isIdempotent() async throws {
        let recorder = CallRecorder()
        let store = makeStore(chores: [chore("c1")],
                              delay: .milliseconds(10),
                              recorder: recorder)

        store.startCompletion(choreID: "c1")
        store.startCompletion(choreID: "c1")
        try await Task.sleep(for: .milliseconds(80))

        let calls = await recorder.calls
        #expect(calls == ["c1"])
    }

    @Test("startCompletion restores chore + records failure on API error")
    func startCompletion_apiFailure_restoresChore() async throws {
        let recorder = CallRecorder()
        await recorder.setErrorToThrow(TestError.simulated)

        let store = makeStore(chores: [chore("c0"), chore("c1"), chore("c2")],
                              delay: .milliseconds(10),
                              recorder: recorder)

        store.startCompletion(choreID: "c1")
        try await Task.sleep(for: .milliseconds(80))

        #expect(store.chores.map(\.id) == ["c0", "c1", "c2"])
        #expect(store.pendingCompletions.contains("c1") == false)
        if case .failed = store.phase { } else {
            Issue.record("expected phase .failed, got \(store.phase)")
        }
    }
}

// MARK: - Test support

actor CallRecorder {
    private(set) var calls: [String] = []
    private var errorToThrow: Error?

    func setErrorToThrow(_ error: Error) {
        self.errorToThrow = error
    }

    func record(_ id: String) throws {
        if let errorToThrow {
            throw errorToThrow
        }
        calls.append(id)
    }
}

enum TestError: Error {
    case simulated
}
