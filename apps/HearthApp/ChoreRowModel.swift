import Foundation

/// Role of a chore relative to the current user, drives the row's left-side
/// ring/avatar slot.
enum ChoreRole {
    case mine
    case open
    case other
}

func role(for chore: HearthChore, me: String?) -> ChoreRole {
    guard let claimant = chore.claimedBy else { return .open }
    if claimant == me { return .mine }
    return .other
}

enum ChipKind: Equatable {
    case overdue
    case tonight
    case scheduled
}

struct Chip: Equatable {
    let text: String
    let kind: ChipKind
}

/// Chip rendered beneath a chore title when there's time pressure. Three
/// states, in priority order:
/// - `.overdue` → peach "N day(s) late"
/// - `.tonight` → amber "Tonight" (willCompleteBy is same local day)
/// - `.scheduled` → muted "by Tomorrow" / "by Fri" / "by Oct 30"
/// Returns nil when there's no time pressure to surface.
func chipFor(
    overdue: Bool,
    willCompleteBy: Date?,
    now: Date,
    calendar: Calendar = .current
) -> Chip? {
    if overdue, let due = willCompleteBy {
        let seconds = now.timeIntervalSince(due)
        let daysLate = max(1, Int(ceil(seconds / 86_400)))
        let text = daysLate == 1 ? "1 day late" : "\(daysLate) days late"
        return Chip(text: text, kind: .overdue)
    }

    guard let due = willCompleteBy else { return nil }

    if calendar.isDate(due, inSameDayAs: now) {
        return Chip(text: "Tonight", kind: .tonight)
    }

    let startOfToday = calendar.startOfDay(for: now)
    let startOfDue = calendar.startOfDay(for: due)
    let dayDelta = calendar.dateComponents([.day], from: startOfToday, to: startOfDue).day ?? 0

    if dayDelta == 1 {
        return Chip(text: "by Tomorrow", kind: .scheduled)
    }

    if dayDelta >= 2 && dayDelta <= 6 {
        let weekday = due.formatted(.dateTime.weekday(.abbreviated))
        return Chip(text: "by \(weekday)", kind: .scheduled)
    }

    let monthDay = due.formatted(.dateTime.month(.abbreviated).day())
    return Chip(text: "by \(monthDay)", kind: .scheduled)
}
