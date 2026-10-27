import Foundation
import Testing
@testable import HearthApp

@MainActor
@Suite("BacklogStore create")
struct BacklogStoreCreateTests {

    private func makeStore(
        recorder: CreateRecorder = CreateRecorder(),
        result: HearthChore? = nil
    ) -> BacklogStore {
        let returned = result ?? Self.chore(id: "new", title: "x")
        return BacklogStore(
            session: { "token" },
            createCommit: { _, title, notes in
                try await recorder.record(title: title, notes: notes)
                return HearthChore(
                    id: returned.id,
                    listId: returned.listId,
                    creatorId: returned.creatorId,
                    title: title,
                    notes: notes,
                    state: returned.state,
                    claimedBy: returned.claimedBy,
                    claimedAt: returned.claimedAt,
                    willCompleteBy: returned.willCompleteBy,
                    completedBy: returned.completedBy,
                    completedAt: returned.completedAt,
                    createdAt: returned.createdAt,
                    updatedAt: returned.updatedAt,
                    overdue: returned.overdue
                )
            }
        )
    }

    private static func chore(id: String, title: String) -> HearthChore {
        HearthChore(
            id: id, listId: "l", creatorId: "me", title: title,
            notes: nil, state: "open",
            claimedBy: nil, claimedAt: nil, willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "", updatedAt: "", overdue: false
        )
    }

    @Test("create trims title whitespace")
    func create_trimsTitleWhitespace() async {
        let recorder = CreateRecorder()
        let store = makeStore(recorder: recorder)

        await store.create(title: "  Descale the kettle  ", notes: nil)

        let calls = await recorder.calls
        #expect(calls.count == 1)
        #expect(calls.first?.title == "Descale the kettle")
    }

    @Test("create drops empty notes")
    func create_dropsEmptyNotes() async {
        let recorder = CreateRecorder()
        let store = makeStore(recorder: recorder)

        await store.create(title: "Mop the kitchen", notes: "   ")

        let calls = await recorder.calls
        #expect(calls.first?.notes == nil)
    }

    @Test("create keeps non-empty notes trimmed")
    func create_keepsTrimmedNotes() async {
        let recorder = CreateRecorder()
        let store = makeStore(recorder: recorder)

        await store.create(title: "Water the plants", notes: "  front porch  ")

        let calls = await recorder.calls
        #expect(calls.first?.notes == "front porch")
    }

    @Test("create prepends new chore on success")
    func create_prependsOnSuccess() async {
        let existing = Self.chore(id: "c1", title: "existing")
        let store = makeStore(result: Self.chore(id: "cnew", title: "fresh"))
        store.setChoresForTesting([existing])

        await store.create(title: "fresh", notes: nil)

        #expect(store.chores.map(\.id) == ["cnew", "c1"])
    }

    @Test("create surfaces failure in phase")
    func create_surfacesFailureInPhase() async {
        let recorder = CreateRecorder()
        await recorder.setErrorToThrow(TestError.simulated)
        let store = makeStore(recorder: recorder)

        await store.create(title: "x", notes: nil)

        if case .failed = store.phase { } else {
            Issue.record("expected phase .failed, got \(store.phase)")
        }
        #expect(store.chores.isEmpty)
    }
}

actor CreateRecorder {
    struct Call: Equatable {
        let title: String
        let notes: String?
    }

    private(set) var calls: [Call] = []
    private var errorToThrow: Error?

    func setErrorToThrow(_ error: Error) {
        self.errorToThrow = error
    }

    func record(title: String, notes: String?) throws {
        if let errorToThrow { throw errorToThrow }
        calls.append(Call(title: title, notes: notes))
    }
}
