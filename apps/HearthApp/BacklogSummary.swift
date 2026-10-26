import Foundation

/// Returns the one-line summary that sits under the "Backlog" title in the
/// header. Speaks to the current user's own load — "N of yours" + "K of yours
/// late". The partner's late items are intentionally not counted; they show up
/// as peach chips in the Claimed section.
///
/// Rules:
/// - 0 mine → "Nothing on your plate. Lovely."
/// - N mine, 0 of yours late → "N things are yours. Nice and calm."
/// - N mine, 1 of yours late → "N things are yours, one running a little late."
/// - N mine, K of yours late → "N things are yours, K running a little late."
///
/// Counts render as words for 1–5, digits for 6+.
func summaryLine(chores: [HearthChore], currentUserID: String?) -> String {
    guard let me = currentUserID else { return "Nothing on your plate. Lovely." }
    let mine = chores.filter { $0.claimedBy == me }
    let mineCount = mine.count
    let lateCount = mine.filter { $0.overdue }.count

    if mineCount == 0 { return "Nothing on your plate. Lovely." }

    let mineClause: String
    if mineCount == 1 {
        mineClause = "One thing is yours"
    } else {
        mineClause = "\(countWord(mineCount, capitalized: true)) things are yours"
    }

    let tail: String
    if lateCount == 0 {
        tail = ". Nice and calm."
    } else if lateCount == 1 {
        tail = ", one running a little late."
    } else {
        tail = ", \(countWord(lateCount, capitalized: false)) running a little late."
    }

    return mineClause + tail
}

private func countWord(_ n: Int, capitalized: Bool) -> String {
    switch n {
    case 1: return capitalized ? "One" : "one"
    case 2: return capitalized ? "Two" : "two"
    case 3: return capitalized ? "Three" : "three"
    case 4: return capitalized ? "Four" : "four"
    case 5: return capitalized ? "Five" : "five"
    default: return String(n)
    }
}
