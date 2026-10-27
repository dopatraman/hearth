import Foundation
import Testing
@testable import HearthApp

@Suite("groupedSections")
struct BacklogSectionsTests {
    private let me = "u_self"
    private let partner = "u_maya"

    private func chore(
        id: String,
        claimedBy: String? = nil,
        overdue: Bool = false,
        willCompleteBy: String? = nil,
        createdAt: String = "2026-10-20T10:00:00.000Z"
    ) -> HearthChore {
        HearthChore(
            id: id,
            listId: "l",
            creatorId: me,
            title: id,
            notes: nil,
            state: "open",
            claimedBy: claimedBy,
            claimedAt: claimedBy == nil ? nil : "2026-10-20T10:00:00.000Z",
            willCompleteBy: willCompleteBy,
            completedBy: nil,
            completedAt: nil,
            createdAt: createdAt,
            updatedAt: createdAt,
            overdue: overdue
        )
    }

    @Test("all filter → both sections")
    func groupedSections_allFilter_returnsClaimedAndUpForGrabs() {
        let chores = [
            chore(id: "c1", claimedBy: me),
            chore(id: "c2", claimedBy: nil),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.map(\.kind) == [.claimed, .upForGrabs])
    }

    @Test("member filter → up-for-grabs hidden")
    func groupedSections_memberFilter_hidesUpForGrabs() {
        let chores = [
            chore(id: "c1", claimedBy: me),
            chore(id: "c2", claimedBy: nil),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .member(me))
        #expect(sections.map(\.kind) == [.claimed])
    }

    @Test("member filter → claimed includes only that member's chores")
    func groupedSections_memberFilter_claimedIncludesOnlyThatMember() {
        let chores = [
            chore(id: "c1", claimedBy: me),
            chore(id: "c2", claimedBy: partner),
            chore(id: "c3", claimedBy: nil),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .member(partner))
        #expect(sections.first?.items.map(\.id) == ["c2"])
    }

    @Test("member filter with no matches → empty sections")
    func groupedSections_memberFilterNoMatches_returnsEmpty() {
        let chores = [chore(id: "c1", claimedBy: nil)]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .member(partner))
        #expect(sections.isEmpty)
    }

    @Test("claimed section → late sorts above on-time")
    func groupedSections_claimedSection_lateSortsAbove_onTime() {
        let chores = [
            chore(id: "on-time", claimedBy: me, overdue: false),
            chore(id: "late", claimedBy: me, overdue: true),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.first?.items.map(\.id) == ["late", "on-time"])
    }

    @Test("claimed section → within late, earlier due sorts first")
    func groupedSections_claimedSection_withinLate_earlierDueSortsFirst() {
        let chores = [
            chore(id: "later", claimedBy: me, overdue: true,
                  willCompleteBy: "2026-10-24T23:59:59.000Z"),
            chore(id: "earlier", claimedBy: me, overdue: true,
                  willCompleteBy: "2026-10-20T23:59:59.000Z"),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.first?.items.map(\.id) == ["earlier", "later"])
    }

    @Test("claimed section → tie-break by createdAt ascending")
    func groupedSections_claimedSection_tieBreakByCreatedAt() {
        let chores = [
            chore(id: "newer", claimedBy: me, overdue: false,
                  willCompleteBy: nil,
                  createdAt: "2026-10-20T10:00:00.000Z"),
            chore(id: "older", claimedBy: me, overdue: false,
                  willCompleteBy: nil,
                  createdAt: "2026-10-18T10:00:00.000Z"),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.first?.items.map(\.id) == ["older", "newer"])
    }

    @Test("empty chores → empty array")
    func groupedSections_emptyChores_returnsEmptyArray() {
        let sections = groupedSections(chores: [], currentUserID: me, filter: .all)
        #expect(sections.isEmpty)
    }

    @Test("only unclaimed under .all → only Up for grabs returned")
    func groupedSections_onlyUnclaimed_returnsOnlyUpForGrabs() {
        let chores = [chore(id: "c1", claimedBy: nil)]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.map(\.kind) == [.upForGrabs])
    }

    @Test("willCompleteBy nil sorts after dated among non-late")
    func groupedSections_claimedSection_nullDueDate_sortsAfterDated() {
        let chores = [
            chore(id: "no-date", claimedBy: me, overdue: false, willCompleteBy: nil,
                  createdAt: "2026-10-10T00:00:00.000Z"),
            chore(id: "dated", claimedBy: me, overdue: false,
                  willCompleteBy: "2026-10-28T23:59:59.000Z",
                  createdAt: "2026-10-20T00:00:00.000Z"),
        ]
        let sections = groupedSections(chores: chores, currentUserID: me, filter: .all)
        #expect(sections.first?.items.map(\.id) == ["dated", "no-date"])
    }
}
