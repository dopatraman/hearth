import Foundation

/// "Since" line rendered inside a chore row's expanded drawer. Reads naturally
/// in two shapes depending on claim state:
///
/// - **Claimed** → "You claimed this today" / "Maya claimed this Wed" /
///   "Someone claimed this Oct 15". Uses weekday-short style within a week.
/// - **Unclaimed** → "Added by you · today" / "Added by Maya · 2 days ago" /
///   "Added by someone · Oct 15". Uses relative-duration style.
///
/// Pure function; `now` is injected for deterministic tests. Returns an empty
/// string if neither claimedAt nor createdAt can be parsed.
func sinceLine(
    chore: HearthChore,
    members: [HearthMember],
    currentUserID: String?,
    now: Date,
    calendar: Calendar = .current
) -> String {
    if let claimant = chore.claimedBy, let claimedAt = chore.claimedAt {
        let who = subjectForClaimed(userID: claimant, me: currentUserID, members: members)
        let date = parseDate(claimedAt) ?? now
        let when = claimedWhen(date, now: now, calendar: calendar)
        return "\(who) claimed this \(when)"
    }

    let who = subjectForAdded(userID: chore.creatorId, me: currentUserID, members: members)
    let date = parseDate(chore.createdAt) ?? now
    let when = addedWhen(date, now: now, calendar: calendar)
    return "Added by \(who) · \(when)"
}

private func subjectForClaimed(
    userID: String,
    me: String?,
    members: [HearthMember]
) -> String {
    if let me, userID == me { return "You" }
    return MembersStore.displayName(for: userID, in: members) ?? "Someone"
}

private func subjectForAdded(
    userID: String,
    me: String?,
    members: [HearthMember]
) -> String {
    if let me, userID == me { return "you" }
    return MembersStore.displayName(for: userID, in: members) ?? "someone"
}

/// Claimed style: when it happened. "today" → weekday abbreviated → short date.
private func claimedWhen(_ date: Date, now: Date, calendar: Calendar) -> String {
    let dayDelta = calendar.dateComponents(
        [.day],
        from: calendar.startOfDay(for: date),
        to: calendar.startOfDay(for: now)
    ).day ?? 0

    if dayDelta == 0 { return "today" }
    if dayDelta >= 1 && dayDelta <= 6 {
        return date.formatted(.dateTime.weekday(.abbreviated))
    }
    return date.formatted(.dateTime.month(.abbreviated).day())
}

/// Added style: how long ago. "today" / "yesterday" / "N days ago" / short date.
private func addedWhen(_ date: Date, now: Date, calendar: Calendar) -> String {
    let dayDelta = calendar.dateComponents(
        [.day],
        from: calendar.startOfDay(for: date),
        to: calendar.startOfDay(for: now)
    ).day ?? 0

    switch dayDelta {
    case 0: return "today"
    case 1: return "yesterday"
    case 2...6: return "\(dayDelta) days ago"
    default: return date.formatted(.dateTime.month(.abbreviated).day())
    }
}

private func parseDate(_ s: String) -> Date? {
    ISO8601DateFormatter.hearthISO().date(from: s) ?? ISO8601DateFormatter().date(from: s)
}
