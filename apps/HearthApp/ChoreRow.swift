import SwiftUI

/// Rounded white card representation of a single `HearthChore` — closed state
/// only. The left-side slot carries semantic role (mine / open / other). The
/// title stacks over an optional chip when there's time pressure. A trailing
/// chevron hints the row will expand (expand drawer lands in Slice 6).
struct ChoreRow: View {
    let chore: HearthChore
    let currentUserID: String?
    let members: [HearthMember]
    let onComplete: () -> Void

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
        HStack(spacing: 4) {
            leftSlot

            VStack(alignment: .leading, spacing: 5) {
                Text(chore.title)
                    .font(.hearthBody(size: 16, weight: .medium))
                    .foregroundStyle(Color.hearthTextPrimary)
                    .lineLimit(2)

                if let chip {
                    ChipView(chip: chip)
                }
            }
            .padding(.leading, 4)
            .frame(maxWidth: .infinity, alignment: .leading)

            Image(systemName: "chevron.down")
                .font(.system(size: 14, weight: .semibold))
                .foregroundStyle(Color.hearthChevron)
                .frame(width: 24)
        }
        .padding(.leading, 6)
        .padding(.trailing, 12)
        .padding(.vertical, 8)
        .frame(minHeight: 56)
        .background(
            RoundedRectangle(cornerRadius: 22, style: .continuous)
                .fill(Color.hearthCardBackground)
                .shadow(color: .black.opacity(0.04), radius: 2, y: 1)
                .shadow(color: .black.opacity(0.04), radius: 18, y: 6)
        )
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
}

extension Color {
    /// White card background — pure white in light, warm-dark in dark.
    static let hearthCardBackground = Color(
        light: .white,
        dark: Color(red: 0x2A / 255, green: 0x27 / 255, blue: 0x24 / 255)
    )

    /// Muted chevron tint.
    static let hearthChevron = Color(
        light: Color(red: 0x9E / 255, green: 0x9A / 255, blue: 0xAF / 255),
        dark: Color(red: 0x7E / 255, green: 0x7A / 255, blue: 0x8F / 255)
    )
}

private extension Color {
    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark ? UIColor(dark) : UIColor(light)
        })
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
                chore: previewChore(title: "Clean out the fridge", claimedBy: "me",
                                    willCompleteBy: "2026-10-23T23:59:59.000Z", overdue: true),
                currentUserID: "me",
                members: members,
                onComplete: {}
            )
            ChoreRow(
                chore: previewChore(title: "Take out the bins", claimedBy: "me",
                                    willCompleteBy: "2026-10-25T23:59:59.000Z"),
                currentUserID: "me",
                members: members,
                onComplete: {}
            )
            ChoreRow(
                chore: previewChore(title: "Water the plants", claimedBy: "me"),
                currentUserID: "me",
                members: members,
                onComplete: {}
            )
            ChoreRow(
                chore: previewChore(title: "Mop the kitchen", claimedBy: "u_maya",
                                    willCompleteBy: "2026-10-26T23:59:59.000Z"),
                currentUserID: "me",
                members: members,
                onComplete: {}
            )
            ChoreRow(
                chore: previewChore(title: "Wipe down the stove", claimedBy: nil),
                currentUserID: "me",
                members: members,
                onComplete: {}
            )
        }
        .padding(16)
    }
    .background(Color.hearthBackground)
}

private func previewChore(
    title: String,
    claimedBy: String? = nil,
    willCompleteBy: String? = nil,
    overdue: Bool = false
) -> HearthChore {
    HearthChore(
        id: UUID().uuidString,
        listId: "l",
        creatorId: "me",
        title: title,
        notes: nil,
        state: "open",
        claimedBy: claimedBy,
        claimedAt: nil,
        willCompleteBy: willCompleteBy,
        completedBy: nil,
        completedAt: nil,
        createdAt: "2026-10-20T10:00:00.000Z",
        updatedAt: "2026-10-20T10:00:00.000Z",
        overdue: overdue
    )
}
#endif
