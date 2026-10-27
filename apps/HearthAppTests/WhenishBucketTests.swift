import Foundation
import Testing
@testable import HearthApp

@Suite("WhenishBucket")
struct WhenishBucketTests {

    private var calendar: Calendar {
        var cal = Calendar(identifier: .gregorian)
        cal.timeZone = TimeZone(identifier: "America/Los_Angeles")!
        return cal
    }

    private func date(_ y: Int, _ m: Int, _ d: Int, _ h: Int = 12) -> Date {
        var comps = DateComponents()
        comps.year = y; comps.month = m; comps.day = d; comps.hour = h; comps.minute = 0
        comps.timeZone = TimeZone(identifier: "America/Los_Angeles")
        return Calendar(identifier: .gregorian).date(from: comps)!
    }

    @Test("someday resolves to nil")
    func someday_resolvesToNil() {
        let now = date(2026, 10, 26)
        #expect(WhenishBucket.someday.resolve(now: now, calendar: calendar) == nil)
    }

    @Test("tonight resolves to end of today")
    func tonight_endOfToday() {
        let now = date(2026, 10, 26, 15)
        let resolved = WhenishBucket.tonight.resolve(now: now, calendar: calendar)
        #expect(resolved != nil)
        let comps = calendar.dateComponents([.year, .month, .day, .hour], from: resolved!)
        #expect(comps.year == 2026 && comps.month == 10 && comps.day == 26)
        #expect(comps.hour == 23)
    }

    @Test("tomorrow resolves to end of next day")
    func tomorrow_endOfNextDay() {
        let now = date(2026, 10, 26, 9)
        let resolved = WhenishBucket.tomorrow.resolve(now: now, calendar: calendar)
        let comps = calendar.dateComponents([.year, .month, .day, .hour], from: resolved!)
        #expect(comps.year == 2026 && comps.month == 10 && comps.day == 27)
        #expect(comps.hour == 23)
    }

    @Test("weekend from a Monday resolves to the next Sunday")
    func weekend_fromMonday_resolvesToSunday() {
        // Mon 2026-10-26 → Sun 2026-11-01
        let now = date(2026, 10, 26, 12)
        let resolved = WhenishBucket.weekend.resolve(now: now, calendar: calendar)
        let comps = calendar.dateComponents([.year, .month, .day, .weekday], from: resolved!)
        #expect(comps.year == 2026 && comps.month == 11 && comps.day == 1)
        #expect(comps.weekday == 1)
    }

    @Test("weekend from Sunday resolves to the same Sunday")
    func weekend_fromSunday_resolvesToSameSunday() {
        // Sun 2026-11-01
        let now = date(2026, 11, 1, 10)
        let resolved = WhenishBucket.weekend.resolve(now: now, calendar: calendar)
        let comps = calendar.dateComponents([.year, .month, .day, .weekday], from: resolved!)
        #expect(comps.year == 2026 && comps.month == 11 && comps.day == 1)
        #expect(comps.weekday == 1)
    }

    @Test("toastSuffix branches")
    func toastSuffix_branches() {
        #expect(WhenishBucket.tonight.toastSuffix == " by tonight")
        #expect(WhenishBucket.tomorrow.toastSuffix == " by tomorrow")
        #expect(WhenishBucket.weekend.toastSuffix == " by the weekend")
        #expect(WhenishBucket.someday.toastSuffix == "")
    }
}
