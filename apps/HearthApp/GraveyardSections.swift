import Foundation

/// A day-bucket in the Done tab. `label` is the human-facing header
/// ("Today", "Yesterday", a weekday name, or "MMM d").
struct GraveyardGroup: Identifiable, Equatable {
    let id: String
    let label: String
    let items: [HearthChore]
}

/// Groups completed chores into day buckets ordered newest first. Items
/// within a bucket are also ordered newest first by `completedAt`. Chores
/// with no (or unparseable) `completedAt` are dropped.
///
/// Bucket labels:
/// - today → "Today"
/// - yesterday → "Yesterday"
/// - 2–6 days back → weekday name (e.g. "Thursday")
/// - 7+ days back → "MMM d" (e.g. "Oct 20")
func groupedGraveyard(
    chores: [HearthChore],
    now: Date,
    calendar: Calendar = .current
) -> [GraveyardGroup] {
    let today = calendar.startOfDay(for: now)
    let isoParser = ISO8601DateFormatter.hearthISO()
    let fallbackParser = ISO8601DateFormatter()

    struct DayBucket {
        let dayStart: Date
        let daysBack: Int
        var items: [(chore: HearthChore, completedAt: Date)] = []
    }

    var buckets: [Date: DayBucket] = [:]
    for chore in chores {
        guard let completedAtString = chore.completedAt else { continue }
        guard let date = isoParser.date(from: completedAtString)
            ?? fallbackParser.date(from: completedAtString) else { continue }
        let dayStart = calendar.startOfDay(for: date)
        let daysBack = calendar.dateComponents([.day], from: dayStart, to: today).day ?? 0

        if buckets[dayStart] == nil {
            buckets[dayStart] = DayBucket(dayStart: dayStart, daysBack: daysBack)
        }
        buckets[dayStart]?.items.append((chore, date))
    }

    let sortedBuckets = buckets.values.sorted { $0.dayStart > $1.dayStart }

    let weekdayFormatter = DateFormatter()
    weekdayFormatter.calendar = calendar
    weekdayFormatter.timeZone = calendar.timeZone
    weekdayFormatter.dateFormat = "EEEE"

    let monthDayFormatter = DateFormatter()
    monthDayFormatter.calendar = calendar
    monthDayFormatter.timeZone = calendar.timeZone
    monthDayFormatter.dateFormat = "MMM d"

    return sortedBuckets.map { bucket in
        let label: String
        switch bucket.daysBack {
        case 0: label = "Today"
        case 1: label = "Yesterday"
        case 2...6: label = weekdayFormatter.string(from: bucket.dayStart)
        default: label = monthDayFormatter.string(from: bucket.dayStart)
        }
        let items = bucket.items
            .sorted { $0.completedAt > $1.completedAt }
            .map(\.chore)
        let id = ISO8601DateFormatter().string(from: bucket.dayStart)
        return GraveyardGroup(id: id, label: label, items: items)
    }
}
