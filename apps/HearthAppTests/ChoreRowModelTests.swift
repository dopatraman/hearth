import Foundation
import Testing
@testable import HearthApp

@Suite("role")
struct ChoreRoleTests {
    private let me = "u_self"
    private let partner = "u_maya"

    private func chore(claimedBy: String?) -> HearthChore {
        HearthChore(
            id: "x", listId: "l", creatorId: me, title: "x",
            notes: nil, state: "open",
            claimedBy: claimedBy, claimedAt: nil, willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "", updatedAt: "", overdue: false
        )
    }

    @Test("claimed by me → .mine")
    func role_claimedByMe_isMine() {
        #expect(role(for: chore(claimedBy: me), me: me) == .mine)
    }

    @Test("unclaimed → .open")
    func role_unclaimed_isOpen() {
        #expect(role(for: chore(claimedBy: nil), me: me) == .open)
    }

    @Test("claimed by other → .other")
    func role_claimedByOther_isOther() {
        #expect(role(for: chore(claimedBy: partner), me: me) == .other)
    }

    @Test("nil me, claimed → .other (safe default)")
    func role_nilMe_claimed_isOther() {
        #expect(role(for: chore(claimedBy: partner), me: nil) == .other)
    }
}

@Suite("chipFor")
struct ChipForTests {
    /// Fixed "now" so date math is deterministic: 2026-10-25 (Saturday) 10:00 local.
    private let now: Date = {
        var comps = DateComponents()
        comps.year = 2026
        comps.month = 10
        comps.day = 25
        comps.hour = 10
        return Calendar.current.date(from: comps)!
    }()

    private func date(y: Int, m: Int, d: Int, h: Int = 23, min: Int = 59) -> Date {
        var comps = DateComponents()
        comps.year = y; comps.month = m; comps.day = d
        comps.hour = h; comps.minute = min
        return Calendar.current.date(from: comps)!
    }

    @Test("overdue 1 day singular → '1 day late'")
    func chip_overdueOneDay_singular_returnsOneDayLate() {
        let due = date(y: 2026, m: 10, d: 24) // ~16h ago
        let chip = chipFor(overdue: true, willCompleteBy: due, now: now)
        #expect(chip == Chip(text: "1 day late", kind: .overdue))
    }

    @Test("overdue 2 days plural → '2 days late'")
    func chip_overdueTwoDays_plural_returnsTwoDaysLate() {
        let due = date(y: 2026, m: 10, d: 23) // ~1.6 days ago
        let chip = chipFor(overdue: true, willCompleteBy: due, now: now)
        #expect(chip == Chip(text: "2 days late", kind: .overdue))
    }

    @Test("overdue 25h → rounds up to '2 days late'")
    func chip_overdueTwentyFiveHours_roundsUpToTwoDaysLate() {
        // 25 hours before now = 2026-10-24 09:00
        let due = Calendar.current.date(byAdding: .hour, value: -25, to: now)!
        let chip = chipFor(overdue: true, willCompleteBy: due, now: now)
        #expect(chip == Chip(text: "2 days late", kind: .overdue))
    }

    @Test("same local day → 'Tonight'")
    func chip_dueTodaySameLocalDay_returnsTonight() {
        let due = date(y: 2026, m: 10, d: 25) // today end-of-day
        let chip = chipFor(overdue: false, willCompleteBy: due, now: now)
        #expect(chip == Chip(text: "Tonight", kind: .tonight))
    }

    @Test("tomorrow → 'by Tomorrow'")
    func chip_dueTomorrow_returnsByTomorrow() {
        let due = date(y: 2026, m: 10, d: 26)
        let chip = chipFor(overdue: false, willCompleteBy: due, now: now)
        #expect(chip == Chip(text: "by Tomorrow", kind: .scheduled))
    }

    @Test("within 6 days → 'by <weekday>'")
    func chip_dueInSixDays_returnsByWeekdayShort() {
        // 2026-10-31 is a Saturday
        let due = date(y: 2026, m: 10, d: 31)
        let chip = chipFor(overdue: false, willCompleteBy: due, now: now)
        #expect(chip?.kind == .scheduled)
        #expect(chip?.text.hasPrefix("by ") == true)
        // Weekday short name — e.g. "Sat"
        #expect((chip?.text.count ?? 0) == 6) // "by Sat"
    }

    @Test("beyond 6 days → 'by <month> <day>'")
    func chip_dueInTwoWeeks_returnsByMonthDay() {
        let due = date(y: 2026, m: 11, d: 10)
        let chip = chipFor(overdue: false, willCompleteBy: due, now: now)
        #expect(chip?.kind == .scheduled)
        #expect(chip?.text.hasPrefix("by Nov ") == true)
    }

    @Test("no due date, not overdue → nil")
    func chip_noDate_returnsNil() {
        #expect(chipFor(overdue: false, willCompleteBy: nil, now: now) == nil)
    }

    @Test("overdue trumps future chip")
    func chip_overdueTrumpsFutureChip() {
        // overdue=true wins even if the date is in the future (shouldn't
        // happen in practice but lock the rule)
        let due = date(y: 2026, m: 10, d: 26)
        let chip = chipFor(overdue: true, willCompleteBy: due, now: now)
        #expect(chip?.kind == .overdue)
    }
}
