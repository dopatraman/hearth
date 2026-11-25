import SwiftUI

enum BacklogSectionKind: Hashable {
    case recurring
    case claimed
    case upForGrabs
    case snoozed
}

struct BacklogSection: Identifiable, Equatable {
    let kind: BacklogSectionKind
    let label: String
    let dotColor: Color
    let items: [HearthChore]

    var id: BacklogSectionKind { kind }
}

/// Groups the current chore list into the two sections the mockup shows:
/// **Claimed** (anyone's active claims, late items sorted on top) and
/// **Up for grabs** (unclaimed). Under a `.member(id)` filter, Claimed is
/// narrowed to that household member and Up for grabs is omitted entirely.
///
/// Pure function — sections with no items are omitted so the view can
/// `ForEach` without extra guards.
///
/// Claimed sort key: `(overdue DESC, willCompleteBy ASC NULLS LAST, createdAt ASC)`.
func groupedSections(
    chores: [HearthChore],
    currentUserID: String?,
    filter: BacklogFilter
) -> [BacklogSection] {
    let active = chores.filter { $0.snoozedUntil == nil }
    let snoozedPool = chores.filter { $0.snoozedUntil != nil }

    // Unclaimed recurring rows feed the rail; claimed recurring rows fall
    // through to the Claimed section (rendered with a "Recurring" label).
    // Only shown under .all — filtering by a member hides the rail since the
    // rail has no concept of "claimed by".
    let recurringRail: [HearthChore]
    switch filter {
    case .all:
        recurringRail = active
            .filter { $0.isRecurring && $0.claimedBy == nil }
            .sorted { $0.createdAt < $1.createdAt }
    case .member:
        recurringRail = []
    }

    let claimedPool: [HearthChore]
    switch filter {
    case .all:
        claimedPool = active.filter { $0.claimedBy != nil }
    case .member(let memberID):
        claimedPool = active.filter { $0.claimedBy == memberID }
    }
    let claimed = claimedPool.sorted(by: claimedSectionSort)

    var sections: [BacklogSection] = []
    if !recurringRail.isEmpty {
        sections.append(
            BacklogSection(
                kind: .recurring,
                label: "Recurring",
                dotColor: .hearthRecurringAccent,
                items: recurringRail
            )
        )
    }
    if !claimed.isEmpty {
        sections.append(
            BacklogSection(
                kind: .claimed,
                label: "Claimed",
                dotColor: .hearthLavender,
                items: claimed
            )
        )
    }

    if filter == .all {
        let upForGrabs = active
            .filter { $0.claimedBy == nil && !$0.isRecurring }
            .sorted(by: upForGrabsSort)
        if !upForGrabs.isEmpty {
            sections.append(
                BacklogSection(
                    kind: .upForGrabs,
                    label: "Up for grabs",
                    dotColor: .hearthSectionDotMuted,
                    items: upForGrabs
                )
            )
        }
    }

    let snoozed: [HearthChore]
    switch filter {
    case .all:
        snoozed = snoozedPool
    case .member(let memberID):
        snoozed = snoozedPool.filter { $0.claimedBy == memberID }
    }
    if !snoozed.isEmpty {
        let ordered = snoozed.sorted { $0.createdAt < $1.createdAt }
        sections.append(
            BacklogSection(
                kind: .snoozed,
                label: "Snoozed",
                dotColor: .hearthSectionDotMuted,
                items: ordered
            )
        )
    }

    return sections
}

/// Sort predicate for the Claimed section. Leads with the user-controlled
/// `sectionOrder` (nils last), then falls back to overdue → willCompleteBy →
/// createdAt as tiebreakers for the brief window before the server has
/// assigned orders to legacy rows.
private func claimedSectionSort(_ a: HearthChore, _ b: HearthChore) -> Bool {
    switch (a.sectionOrder, b.sectionOrder) {
    case let (lhs?, rhs?) where lhs != rhs:
        return lhs < rhs
    case (nil, _?):
        return false
    case (_?, nil):
        return true
    default:
        break
    }

    if a.overdue != b.overdue { return a.overdue && !b.overdue }

    switch (a.willCompleteBy, b.willCompleteBy) {
    case let (lhs?, rhs?) where lhs != rhs:
        return lhs < rhs
    case (nil, _?):
        return false
    case (_?, nil):
        return true
    default:
        break
    }

    return a.createdAt < b.createdAt
}

/// Sort predicate for the Up-for-grabs section. Leads with `sectionOrder`
/// (nils last), then falls back to `createdAt` as a tiebreaker.
private func upForGrabsSort(_ a: HearthChore, _ b: HearthChore) -> Bool {
    switch (a.sectionOrder, b.sectionOrder) {
    case let (lhs?, rhs?) where lhs != rhs:
        return lhs < rhs
    case (nil, _?):
        return false
    case (_?, nil):
        return true
    default:
        break
    }
    return a.createdAt < b.createdAt
}

extension Color {
    /// Muted gray dot for the "Up for grabs" section header.
    static let hearthSectionDotMuted = Color(
        light: Color(red: 0xC9 / 255, green: 0xC6 / 255, blue: 0xD4 / 255),
        dark: Color(red: 0x6E / 255, green: 0x6A / 255, blue: 0x80 / 255)
    )

    /// Teal accent used for the Recurring section's icon + label.
    static let hearthRecurringAccent = Color(
        light: Color(red: 0x1A / 255, green: 0x5E / 255, blue: 0x54 / 255),
        dark: Color(red: 0x7F / 255, green: 0xC8 / 255, blue: 0xBB / 255)
    )
}

private extension Color {
    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark ? UIColor(dark) : UIColor(light)
        })
    }
}
