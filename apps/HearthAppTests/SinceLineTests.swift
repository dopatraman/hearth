import Foundation
import Testing
@testable import HearthApp

@Suite("sinceLine")
struct SinceLineTests {
    private let me = "u_self"
    private let maya = "u_maya"

    private let members = [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]

    /// Deterministic "now" — Saturday 2026-10-24 10:00 local, matches the
    /// mockup's reference day.
    private let now: Date = {
        var comps = DateComponents()
        comps.year = 2026; comps.month = 10; comps.day = 24
        comps.hour = 10
        return Calendar.current.date(from: comps)!
    }()

    private func iso(_ year: Int, _ month: Int, _ day: Int, hour: Int = 9, min: Int = 0) -> String {
        var comps = DateComponents()
        comps.year = year; comps.month = month; comps.day = day
        comps.hour = hour; comps.minute = min
        let date = Calendar.current.date(from: comps)!
        return ISO8601DateFormatter.hearthISO().string(from: date)
    }

    private func chore(
        creatorId: String = "u_self",
        claimedBy: String? = nil,
        claimedAt: String? = nil,
        createdAt: String
    ) -> HearthChore {
        HearthChore(
            id: "c",
            listId: "l",
            creatorId: creatorId,
            title: "t",
            notes: nil,
            state: "open",
            claimedBy: claimedBy,
            claimedAt: claimedAt,
            willCompleteBy: nil,
            completedBy: nil,
            completedAt: nil,
            createdAt: createdAt,
            updatedAt: createdAt,
            overdue: false
        )
    }

    // --- claimed branch ---

    @Test("claimed by me, same day → 'You claimed this today'")
    func sinceLine_claimedByMe_sameDay() {
        let c = chore(claimedBy: me, claimedAt: iso(2026, 10, 24, hour: 8),
                      createdAt: iso(2026, 10, 24))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "You claimed this today")
    }

    @Test("claimed by me, within week → 'You claimed this <weekday>'")
    func sinceLine_claimedByMe_withinWeek_usesWeekday() {
        // Tue Oct 20
        let c = chore(claimedBy: me, claimedAt: iso(2026, 10, 20),
                      createdAt: iso(2026, 10, 20))
        let out = sinceLine(chore: c, members: members, currentUserID: me, now: now)
        #expect(out == "You claimed this Tue")
    }

    @Test("claimed by other, same day → '<Name> claimed this today'")
    func sinceLine_claimedByOther_sameDay() {
        let c = chore(claimedBy: maya, claimedAt: iso(2026, 10, 24, hour: 8),
                      createdAt: iso(2026, 10, 22))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Maya claimed this today")
    }

    @Test("claimed by other, within week → '<Name> claimed this <weekday>'")
    func sinceLine_claimedByOther_usesName() {
        let c = chore(claimedBy: maya, claimedAt: iso(2026, 10, 21),
                      createdAt: iso(2026, 10, 20))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Maya claimed this Wed")
    }

    @Test("claimed by other, older than 6 days → short date")
    func sinceLine_claimedByOther_older_shortDate() {
        let c = chore(claimedBy: maya, claimedAt: iso(2026, 10, 10),
                      createdAt: iso(2026, 10, 10))
        let out = sinceLine(chore: c, members: members, currentUserID: me, now: now)
        #expect(out.hasPrefix("Maya claimed this Oct ") == true)
    }

    @Test("claimed by unknown user → 'Someone claimed this ...'")
    func sinceLine_claimedByUnknownUser_fallsBackGracefully() {
        let c = chore(claimedBy: "u_ghost", claimedAt: iso(2026, 10, 24, hour: 8),
                      createdAt: iso(2026, 10, 24))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Someone claimed this today")
    }

    // --- unclaimed / added branch ---

    @Test("unclaimed, added by me today → 'Added by you · today'")
    func sinceLine_unclaimed_byMe_today() {
        let c = chore(creatorId: me, claimedBy: nil, claimedAt: nil,
                      createdAt: iso(2026, 10, 24, hour: 8))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Added by you · today")
    }

    @Test("unclaimed, added by me yesterday → 'Added by you · yesterday'")
    func sinceLine_unclaimed_byMe_yesterday() {
        let c = chore(creatorId: me, claimedBy: nil, claimedAt: nil,
                      createdAt: iso(2026, 10, 23))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Added by you · yesterday")
    }

    @Test("unclaimed, added by other 2 days ago → 'Added by <Name> · 2 days ago'")
    func sinceLine_unclaimed_byOther_daysAgo() {
        let c = chore(creatorId: maya, claimedBy: nil, claimedAt: nil,
                      createdAt: iso(2026, 10, 22))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Added by Maya · 2 days ago")
    }

    @Test("unclaimed, older than 6 days → short date")
    func sinceLine_unclaimed_older_shortDate() {
        let c = chore(creatorId: maya, claimedBy: nil, claimedAt: nil,
                      createdAt: iso(2026, 10, 10))
        let out = sinceLine(chore: c, members: members, currentUserID: me, now: now)
        #expect(out.hasPrefix("Added by Maya · Oct ") == true)
    }

    @Test("unclaimed, added by unknown user → 'Added by someone · ...'")
    func sinceLine_unclaimed_byUnknownUser_fallsBackGracefully() {
        let c = chore(creatorId: "u_ghost", claimedBy: nil, claimedAt: nil,
                      createdAt: iso(2026, 10, 24, hour: 8))
        #expect(sinceLine(chore: c, members: members, currentUserID: me, now: now)
                == "Added by someone · today")
    }
}
