import Foundation
import Testing
@testable import HearthApp

@Suite("groupedSections — suggestions")
struct BacklogSectionsSuggestedTests {
    private let me = "u_self"
    private let partner = "u_maya"

    private func chore(
        id: String,
        claimedBy: String? = nil,
        createdAt: String = "2026-10-20T10:00:00.000Z",
        suggestion: ChoreSuggestion? = nil
    ) -> HearthChore {
        HearthChore(
            id: id,
            listId: "l",
            creatorId: me,
            title: id,
            notes: nil,
            state: "open",
            claimedBy: claimedBy,
            claimedAt: claimedBy == nil ? nil : createdAt,
            willCompleteBy: nil,
            completedBy: nil,
            completedAt: nil,
            createdAt: createdAt,
            updatedAt: createdAt,
            overdue: false,
            snoozedUntil: nil,
            suggestion: suggestion
        )
    }

    private func suggestion(
        from: String,
        to: String,
        choreID: String,
        note: String? = nil
    ) -> ChoreSuggestion {
        ChoreSuggestion(
            id: "sug_" + choreID,
            choreID: choreID,
            fromUserID: from,
            toUserID: to,
            note: note,
            createdAt: "2026-10-20T10:00:00.000Z"
        )
    }

    @Test("chore suggested to me → Suggested-for-you section, not Up-for-grabs")
    func suggestedForMe_prependsOwnSection() {
        let suggestedToMe = chore(
            id: "c1",
            suggestion: suggestion(from: partner, to: me, choreID: "c1")
        )
        let plain = chore(id: "c2")
        let sections = groupedSections(
            chores: [suggestedToMe, plain],
            currentUserID: me,
            filter: .all
        )
        #expect(sections.map(\.kind) == [.suggested, .upForGrabs])
        #expect(sections[0].items.map(\.id) == ["c1"])
        #expect(sections[1].items.map(\.id) == ["c2"])
    }

    @Test("no suggestions → Suggested-for-you section hidden")
    func noSuggestions_sectionHidden() {
        let sections = groupedSections(
            chores: [chore(id: "c1"), chore(id: "c2")],
            currentUserID: me,
            filter: .all
        )
        #expect(sections.map(\.kind).contains(.suggested) == false)
    }

    @Test("suggestion to me on a claimed-by-someone chore → still in Suggested section")
    // Chores stay grabbable; the recipient's suggestion ranking wins over
    // claimed presentation. (This is a cross-check that the sectioning rule
    // keys off `.suggestion?.toUserID == me`, not off `claimedBy`.)
    func suggestedForMe_evenIfClaimedByOther() {
        let suggestedButClaimed = chore(
            id: "c1",
            claimedBy: partner,
            suggestion: suggestion(from: partner, to: me, choreID: "c1")
        )
        let sections = groupedSections(
            chores: [suggestedButClaimed],
            currentUserID: me,
            filter: .all
        )
        #expect(sections.first?.kind == .suggested)
        #expect(sections.first?.items.map(\.id) == ["c1"])
    }

    @Test("suggestion I sent → chore stays in Up-for-grabs (not pulled into Suggested)")
    func suggestedByMe_staysInUpForGrabs() {
        let sentByMe = chore(
            id: "c1",
            suggestion: suggestion(from: me, to: partner, choreID: "c1")
        )
        let sections = groupedSections(
            chores: [sentByMe],
            currentUserID: me,
            filter: .all
        )
        #expect(sections.map(\.kind) == [.upForGrabs])
        #expect(sections.first?.items.map(\.id) == ["c1"])
    }

    @Test("suggested chore excluded from Up-for-grabs for the recipient")
    // Guards against double-rendering: a chore the recipient sees in
    // "Suggested for you" must NOT also appear in their Up-for-grabs section.
    func suggestedForMe_excludedFromUpForGrabs() {
        let suggestedToMe = chore(
            id: "c1",
            suggestion: suggestion(from: partner, to: me, choreID: "c1")
        )
        let sections = groupedSections(
            chores: [suggestedToMe],
            currentUserID: me,
            filter: .all
        )
        let upForGrabs = sections.first { $0.kind == .upForGrabs }
        #expect(upForGrabs == nil || upForGrabs?.items.contains(where: { $0.id == "c1" }) == false)
    }

    @Test("Suggested section respects .member filter — hidden under member filter")
    // The member filter is for drilling into one person's claimed workload;
    // "Suggested for you" is personal to the current user and shouldn't leak
    // into another member's filtered view.
    func suggestedForMe_hiddenUnderMemberFilter() {
        let suggestedToMe = chore(
            id: "c1",
            suggestion: suggestion(from: partner, to: me, choreID: "c1")
        )
        let claimedByPartner = chore(id: "c2", claimedBy: partner)
        let sections = groupedSections(
            chores: [suggestedToMe, claimedByPartner],
            currentUserID: me,
            filter: .member(partner)
        )
        #expect(sections.map(\.kind).contains(.suggested) == false)
    }

    @Test("Suggested section is prepended before Claimed and Up-for-grabs")
    func suggestedSection_prependsEverything() {
        let suggestedToMe = chore(
            id: "s",
            suggestion: suggestion(from: partner, to: me, choreID: "s")
        )
        let myClaim = chore(id: "m", claimedBy: me)
        let open = chore(id: "o")
        let sections = groupedSections(
            chores: [suggestedToMe, myClaim, open],
            currentUserID: me,
            filter: .all
        )
        #expect(sections.map(\.kind) == [.suggested, .claimed, .upForGrabs])
    }

    @Test("currentUserID nil → no Suggested section (sign-out edge case)")
    func nilCurrentUser_noSuggestedSection() {
        let suggested = chore(
            id: "c1",
            suggestion: suggestion(from: partner, to: me, choreID: "c1")
        )
        let sections = groupedSections(
            chores: [suggested],
            currentUserID: nil,
            filter: .all
        )
        #expect(sections.map(\.kind).contains(.suggested) == false)
    }
}
