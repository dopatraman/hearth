import SwiftUI

enum BacklogSectionKind: Hashable {
    case claimed
    case upForGrabs
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
/// **Up for grabs** (unclaimed). Under the `.mine` filter, Claimed is
/// narrowed to the current user and Up for grabs is omitted entirely.
///
/// Pure function — sections with no items are omitted so the view can
/// `ForEach` without extra guards.
///
/// Claimed sort key: `(overdue DESC, willCompleteBy ASC NULLS LAST, createdAt ASC)`.
func groupedSections(
    chores: [HearthChore],
    currentUserID: String?,
    filter: BacklogView.Filter
) -> [BacklogSection] {
    let claimedPool: [HearthChore]
    switch filter {
    case .all:
        claimedPool = chores.filter { $0.claimedBy != nil }
    case .mine:
        guard let me = currentUserID else { return [] }
        claimedPool = chores.filter { $0.claimedBy == me }
    }
    let claimed = claimedPool.sorted(by: claimedOrder)

    var sections: [BacklogSection] = []
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
        let upForGrabs = chores
            .filter { $0.claimedBy == nil }
            .sorted { $0.createdAt < $1.createdAt }
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

    return sections
}

/// Sort predicate for the Claimed section:
/// 1. `overdue == true` first.
/// 2. Ascending `willCompleteBy`; nils last.
/// 3. Ascending `createdAt` as tie-break (older first).
private func claimedOrder(_ a: HearthChore, _ b: HearthChore) -> Bool {
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

extension Color {
    /// Muted gray dot for the "Up for grabs" section header.
    static let hearthSectionDotMuted = Color(
        light: Color(red: 0xC9 / 255, green: 0xC6 / 255, blue: 0xD4 / 255),
        dark: Color(red: 0x6E / 255, green: 0x6A / 255, blue: 0x80 / 255)
    )
}

private extension Color {
    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark ? UIColor(dark) : UIColor(light)
        })
    }
}
