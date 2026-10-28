import Foundation
import Testing
@testable import HearthApp

@Suite("groupedGraveyard")
struct GraveyardSectionsTests {

    private var calendar: Calendar {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "America/Los_Angeles")!
        return cal
    }

    private func instant(_ y: Int, _ m: Int, _ d: Int, _ h: Int = 12, _ mi: Int = 0) -> Date {
        var comps = DateComponents()
        comps.year = y; comps.month = m; comps.day = d
        comps.hour = h; comps.minute = mi
        comps.timeZone = TimeZone(identifier: "America/Los_Angeles")
        return Calendar(identifier: .gregorian).date(from: comps)!
    }

    private func iso(_ date: Date) -> String {
        let f = ISO8601DateFormatter()
        f.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
        return f.string(from: date)
    }

    private func chore(id: String, completedAt: Date?) -> HearthChore {
        HearthChore(
            id: id, listId: "l", creatorId: "me",
            title: id, notes: nil, state: "completed",
            claimedBy: "me", claimedAt: nil, willCompleteBy: nil,
            completedBy: "me", completedAt: completedAt.map(iso),
            createdAt: "", updatedAt: "", overdue: false
        )
    }

    // Fixed "now" for all these tests: Tuesday, 2026-10-27 at 15:00 PT.
    private var now: Date { instant(2026, 10, 27, 15) }

    @Test("chore completed earlier today lands in Today")
    func today_bucket() {
        let c = chore(id: "a", completedAt: instant(2026, 10, 27, 9, 30))
        let groups = groupedGraveyard(chores: [c], now: now, calendar: calendar)
        #expect(groups.map(\.label) == ["Today"])
        #expect(groups.first?.items.map(\.id) == ["a"])
    }

    @Test("chore completed yesterday lands in Yesterday")
    func yesterday_bucket() {
        let c = chore(id: "a", completedAt: instant(2026, 10, 26, 20))
        let groups = groupedGraveyard(chores: [c], now: now, calendar: calendar)
        #expect(groups.first?.label == "Yesterday")
    }

    @Test("chore 3 days ago uses weekday label")
    func weekday_bucket() {
        // 2026-10-24 is a Saturday
        let c = chore(id: "a", completedAt: instant(2026, 10, 24, 10))
        let groups = groupedGraveyard(chores: [c], now: now, calendar: calendar)
        #expect(groups.first?.label == "Saturday")
    }

    @Test("chore 7+ days ago uses MMM d label")
    func monthDay_bucket() {
        let c = chore(id: "a", completedAt: instant(2026, 10, 20, 8))
        let groups = groupedGraveyard(chores: [c], now: now, calendar: calendar)
        #expect(groups.first?.label == "Oct 20")
    }

    @Test("groups are ordered newest-first; items within a group newest-first")
    func ordering() {
        let chores = [
            chore(id: "today-am", completedAt: instant(2026, 10, 27, 8)),
            chore(id: "today-pm", completedAt: instant(2026, 10, 27, 14)),
            chore(id: "yesterday", completedAt: instant(2026, 10, 26, 12)),
            chore(id: "last-week", completedAt: instant(2026, 10, 20, 10)),
        ]
        let groups = groupedGraveyard(chores: chores, now: now, calendar: calendar)
        #expect(groups.map(\.label) == ["Today", "Yesterday", "Oct 20"])
        #expect(groups[0].items.map(\.id) == ["today-pm", "today-am"])
    }

    @Test("chore with nil completedAt is dropped")
    func nilCompletedAt_dropped() {
        let c = chore(id: "a", completedAt: nil)
        let groups = groupedGraveyard(chores: [c], now: now, calendar: calendar)
        #expect(groups.isEmpty)
    }

    @Test("empty input returns empty array")
    func empty_input() {
        let groups = groupedGraveyard(chores: [], now: now, calendar: calendar)
        #expect(groups.isEmpty)
    }
}
