import Foundation
import Testing
@testable import HearthApp

@Suite("summaryLine")
struct BacklogSummaryTests {
    private let me = "u_self"
    private let partner = "u_maya"

    private func chore(
        id: String,
        claimedBy: String? = nil,
        overdue: Bool = false
    ) -> HearthChore {
        HearthChore(
            id: id,
            listId: "l",
            creatorId: me,
            title: "t",
            notes: nil,
            state: "open",
            claimedBy: claimedBy,
            claimedAt: nil,
            willCompleteBy: nil,
            completedBy: nil,
            completedAt: nil,
            createdAt: "",
            updatedAt: "",
            overdue: overdue
        )
    }

    @Test("zero mine → nothing on your plate")
    func summaryLine_zeroMine_returnsNothingOnYourPlate() {
        let chores = [
            chore(id: "a", claimedBy: nil),
            chore(id: "b", claimedBy: partner),
        ]
        #expect(summaryLine(chores: chores, currentUserID: me) == "Nothing on your plate. Lovely.")
    }

    @Test("zero mine, partner has late → still nothing on your plate")
    func summaryLine_zeroMine_householdHasLate_stillReturnsNothingOnYourPlate() {
        let chores = [
            chore(id: "a", claimedBy: partner, overdue: true),
            chore(id: "b", claimedBy: partner, overdue: true),
        ]
        #expect(summaryLine(chores: chores, currentUserID: me) == "Nothing on your plate. Lovely.")
    }

    @Test("one mine, zero late → singular + calm suffix")
    func summaryLine_oneMine_zeroLate_singular() {
        let chores = [chore(id: "a", claimedBy: me)]
        #expect(summaryLine(chores: chores, currentUserID: me) == "One thing is yours. Nice and calm.")
    }

    @Test("one mine, one of yours late → singular + singular late")
    func summaryLine_oneMine_oneOfYoursLate_singularLate() {
        let chores = [chore(id: "a", claimedBy: me, overdue: true)]
        #expect(summaryLine(chores: chores, currentUserID: me) == "One thing is yours, one running a little late.")
    }

    @Test("three mine, two of yours late → plural + plural late")
    func summaryLine_threeMine_twoOfYoursLate_pluralLate() {
        let chores = [
            chore(id: "a", claimedBy: me, overdue: true),
            chore(id: "b", claimedBy: me, overdue: true),
            chore(id: "c", claimedBy: me, overdue: false),
        ]
        #expect(summaryLine(chores: chores, currentUserID: me) == "Three things are yours, two running a little late.")
    }

    @Test("three mine, zero of yours late → calm suffix")
    func summaryLine_threeMine_zeroOfYoursLate_calmSuffix() {
        let chores = [
            chore(id: "a", claimedBy: me),
            chore(id: "b", claimedBy: me),
            chore(id: "c", claimedBy: me),
        ]
        #expect(summaryLine(chores: chores, currentUserID: me) == "Three things are yours. Nice and calm.")
    }

    @Test("three mine, partner's late do not count")
    func summaryLine_threeMine_partnersAreLate_doesNotCountThem() {
        let chores = [
            chore(id: "a", claimedBy: me),
            chore(id: "b", claimedBy: me),
            chore(id: "c", claimedBy: me),
            chore(id: "p1", claimedBy: partner, overdue: true),
            chore(id: "p2", claimedBy: partner, overdue: true),
        ]
        #expect(summaryLine(chores: chores, currentUserID: me) == "Three things are yours. Nice and calm.")
    }

    @Test("five mine uses word Five")
    func summaryLine_fiveMine_usesWordFive() {
        let chores = (1...5).map { chore(id: "c\($0)", claimedBy: me) }
        #expect(summaryLine(chores: chores, currentUserID: me) == "Five things are yours. Nice and calm.")
    }

    @Test("six mine falls back to digit")
    func summaryLine_sixMine_fallsBackToDigit() {
        let chores = (1...6).map { chore(id: "c\($0)", claimedBy: me) }
        #expect(summaryLine(chores: chores, currentUserID: me) == "6 things are yours. Nice and calm.")
    }

    @Test("six of yours late falls back to digit in tail")
    func summaryLine_sixLate_fallsBackToDigit() {
        let chores = (1...6).map { chore(id: "c\($0)", claimedBy: me, overdue: true) }
        #expect(summaryLine(chores: chores, currentUserID: me) == "6 things are yours, 6 running a little late.")
    }

    @Test("nil currentUserID → nothing on your plate")
    func summaryLine_nilCurrentUserID_returnsNothingOnYourPlate() {
        let chores = [chore(id: "a", claimedBy: "someone")]
        #expect(summaryLine(chores: chores, currentUserID: nil) == "Nothing on your plate. Lovely.")
    }
}
