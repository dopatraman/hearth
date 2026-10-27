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
            createCommit: { _, title, notes, claim in
                try await recorder.record(title: title, notes: notes, claim: claim)
                return HearthChore(
                    id: returned.id,
                    listId: returned.listId,
                    creatorId: returned.creatorId,
                    title: title,
                    notes: notes,
                    state: returned.state,
                    claimedBy: claim ? returned.creatorId : returned.claimedBy,
                    claimedAt: claim ? "2026-10-26T20:00:00.000Z" : returned.claimedAt,
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

    @Test("create defaults claim to false")
    func create_defaultClaimFalse() async {
        let recorder = CreateRecorder()
        let store = makeStore(recorder: recorder)

        await store.create(title: "x", notes: nil)

        let calls = await recorder.calls
        #expect(calls.first?.claim == false)
    }

    @Test("create with claim passes claim through")
    func create_withClaim_passesClaimThrough() async {
        let recorder = CreateRecorder()
        let store = makeStore(recorder: recorder)

        await store.create(title: "x", notes: nil, claim: true)

        let calls = await recorder.calls
        #expect(calls.first?.claim == true)
    }

    @Test("create with claim surfaces claimed chore")
    func create_withClaim_surfacesClaimedChore() async {
        let store = makeStore(result: Self.chore(id: "cnew", title: "fresh"))

        await store.create(title: "fresh", notes: nil, claim: true)

        #expect(store.chores.first?.claimedBy == "me")
    }

    @Test("successful create resets phase to loaded after a prior failure")
    func create_resetsPhaseToLoadedAfterFailure() async {
        let recorder = CreateRecorder()
        await recorder.setErrorToThrow(TestError.simulated)
        let store = makeStore(recorder: recorder)

        await store.create(title: "x", notes: nil)
        if case .failed = store.phase { } else {
            Issue.record("expected .failed after first create, got \(store.phase)")
        }

        let recorder2 = CreateRecorder()
        let returned = Self.chore(id: "ok", title: "y")
        store.replaceCreateCommitForTesting { _, title, notes, claim in
            try await recorder2.record(title: title, notes: notes, claim: claim)
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

        await store.create(title: "y", notes: nil)

        if case .loaded = store.phase { } else {
            Issue.record("expected .loaded after successful create, got \(store.phase)")
        }
    }
}

actor CreateRecorder {
    struct Call: Equatable {
        let title: String
        let notes: String?
        let claim: Bool
    }

    private(set) var calls: [Call] = []
    private var errorToThrow: Error?

    func setErrorToThrow(_ error: Error) {
        self.errorToThrow = error
    }

    func record(title: String, notes: String?, claim: Bool) throws {
        if let errorToThrow { throw errorToThrow }
        calls.append(Call(title: title, notes: notes, claim: claim))
    }
}
