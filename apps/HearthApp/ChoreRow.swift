import SwiftUI

/// Rounded white card representation of a single `HearthChore`. The closed
/// state shows a ring/avatar slot, title, optional chip, and a chevron. When
/// `isOpen == true` the card expands with a spring to reveal notes, a since
/// line, and action pills tuned to the chore's state.
struct ChoreRow: View {
    let chore: HearthChore
    let currentUserID: String?
    let members: [HearthMember]
    let isOpen: Bool
    let onToggle: () -> Void
    let onComplete: () -> Void
    let onClaim: () -> Void
    let onSetDate: () -> Void
    let onSnooze: () -> Void
    let onRelease: () -> Void

    private var roleValue: ChoreRole {
        role(for: chore, me: currentUserID)
    }

    private var otherInitial: String? {
        guard roleValue == .other, let claimant = chore.claimedBy else { return nil }
        return MembersStore.initial(for: claimant, in: members)
    }

    private var chip: Chip? {
        let dueDate: Date? = chore.willCompleteBy.flatMap { s in
            ISO8601DateFormatter.hearthISO().date(from: s)
                ?? ISO8601DateFormatter().date(from: s)
        }
        return chipFor(overdue: chore.overdue, willCompleteBy: dueDate, now: Date())
    }

    var body: some View {
        VStack(spacing: 0) {
            closedRow
            if isOpen {
                drawer
                    .transition(.opacity.combined(with: .move(edge: .top)))
            }
        }
        .background(
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .fill(Color.hearthSurface)
                .shadow(color: .black.opacity(0.04), radius: 2, y: 1)
                .shadow(color: .black.opacity(0.04), radius: 18, y: 6)
        )
        .animation(.spring(response: 0.4, dampingFraction: 0.82), value: isOpen)
    }

    // MARK: - Closed row

    private var closedRow: some View {
        HStack(spacing: 4) {
            leftSlot

            Button(action: onToggle) {
                VStack(alignment: .leading, spacing: 5) {
                    Text(chore.title)
                        .font(.hearthBody(size: 16, weight: .medium))
                        .foregroundStyle(Color.hearthTextPrimary)
                        .lineLimit(2)
                        .multilineTextAlignment(.leading)

                    if let chip {
                        ChipView(chip: chip)
                    }
                }
                .padding(.leading, 4)
                .frame(maxWidth: .infinity, alignment: .leading)
                .contentShape(Rectangle())
            }
            .buttonStyle(.plain)
            .accessibilityHint(isOpen ? "Collapse" : "Expand")

            Image(systemName: "chevron.down")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Color.hearthChevron)
                .rotationEffect(.degrees(isOpen ? 180 : 0))
                .frame(width: 24)
        }
        .padding(.leading, 6)
        .padding(.trailing, 12)
        .padding(.vertical, 8)
        .frame(minHeight: 56)
    }

    @ViewBuilder
    private var leftSlot: some View {
        if roleValue == .mine {
            Button(action: onComplete) {
                RingSlot(role: roleValue, otherInitial: otherInitial)
            }
            .buttonStyle(.plain)
            .accessibilityLabel("Mark done")
        } else {
            RingSlot(role: roleValue, otherInitial: otherInitial)
                .accessibilityHidden(roleValue == .open)
                .accessibilityLabel(otherLabel)
        }
    }

    private var otherLabel: String {
        guard roleValue == .other, let claimant = chore.claimedBy else { return "" }
        let name = MembersStore.displayName(for: claimant, in: members) ?? "someone"
        return "Claimed by \(name)"
    }

    // MARK: - Drawer

    private var drawer: some View {
        VStack(alignment: .leading, spacing: 14) {
            if let notes = chore.notes, !notes.isEmpty {
                Text(notes)
                    .font(.hearthBody(size: 14))
                    .foregroundStyle(Color.hearthTextSecondary)
                    .fixedSize(horizontal: false, vertical: true)
            }

            Text(sinceLine(
                chore: chore,
                members: members,
                currentUserID: currentUserID,
                now: Date()
            ))
                .font(.hearthBody(size: 12, weight: .medium))
                .foregroundStyle(Color.hearthTextSecondary)

            if !actionPills.isEmpty {
                FlowRow(spacing: 8) {
                    ForEach(actionPills, id: \.label) { pill in
                        ActionPill(label: pill.label, action: pill.action)
                    }
                }
            }
        }
        .padding(.leading, 58)
        .padding(.trailing, 12)
        .padding(.top, 2)
        .padding(.bottom, 10)
    }

    private var actionPills: [(label: String, action: () -> Void)] {
        switch roleValue {
        case .mine:
            return [
                ("Set a date", onSetDate),
                ("Snooze", onSnooze),
                ("Release", onRelease),
            ]
        case .open:
            return [("Claim & pick a day", onClaim)]
        case .other:
            return []
        }
    }
}

// MARK: - Pills

private struct ActionPill: View {
    let label: String
    let action: () -> Void

    var body: some View {
        Button(action: action) {
            Text(label)
                .font(.hearthBody(size: 13, weight: .semibold))
                .foregroundStyle(Color.hearthInsetInk)
                .frame(height: 36)
                .padding(.horizontal, 14)
                .background(Color.hearthInsetFill, in: Capsule())
        }
        .buttonStyle(.plain)
    }
}

/// Minimal pill row: lays action pills out left to right with a trailing
/// Spacer so a single pill still snaps to the leading edge (matching the
/// title above) instead of expanding to fill its row.
private struct FlowRow<Content: View>: View {
    let spacing: CGFloat
    @ViewBuilder let content: Content

    var body: some View {
        HStack(spacing: spacing) {
            content
            Spacer(minLength: 0)
        }
    }
}

#if DEBUG
#Preview {
    let members = [
        HearthMember(id: "me", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]
    return ScrollView {
        VStack(spacing: 10) {
            ChoreRow(
                chore: previewChore(title: "Clean out the fridge",
                                    notes: "Toss anything past its date. Wipe the shelves.",
                                    claimedBy: "me",
                                    willCompleteBy: "2026-10-23T23:59:59.000Z",
                                    overdue: true),
                currentUserID: "me", members: members,
                isOpen: true,
                onToggle: {}, onComplete: {},
                onClaim: {}, onSetDate: {}, onSnooze: {}, onRelease: {}
            )
            ChoreRow(
                chore: previewChore(title: "Wipe down the stove", claimedBy: nil),
                currentUserID: "me", members: members,
                isOpen: true,
                onToggle: {}, onComplete: {},
                onClaim: {}, onSetDate: {}, onSnooze: {}, onRelease: {}
            )
            ChoreRow(
                chore: previewChore(title: "Mop the kitchen", claimedBy: "u_maya",
                                    willCompleteBy: "2026-10-26T23:59:59.000Z"),
                currentUserID: "me", members: members,
                isOpen: true,
                onToggle: {}, onComplete: {},
                onClaim: {}, onSetDate: {}, onSnooze: {}, onRelease: {}
            )
            ChoreRow(
                chore: previewChore(title: "Water the plants", claimedBy: "me"),
                currentUserID: "me", members: members,
                isOpen: false,
                onToggle: {}, onComplete: {},
                onClaim: {}, onSetDate: {}, onSnooze: {}, onRelease: {}
            )
        }
        .padding(16)
    }
    .background(Color.hearthBackground)
}

private func previewChore(
    title: String,
    notes: String? = nil,
    claimedBy: String? = nil,
    willCompleteBy: String? = nil,
    overdue: Bool = false
) -> HearthChore {
    HearthChore(
        id: UUID().uuidString,
        listId: "l",
        creatorId: "me",
        title: title,
        notes: notes,
        state: "open",
        claimedBy: claimedBy,
        claimedAt: claimedBy == nil ? nil : "2026-10-23T14:00:00.000Z",
        willCompleteBy: willCompleteBy,
        completedBy: nil,
        completedAt: nil,
        createdAt: "2026-10-20T10:00:00.000Z",
        updatedAt: "2026-10-20T10:00:00.000Z",
        overdue: overdue
    )
}
#endif
