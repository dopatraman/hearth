import SwiftUI

/// Done-tab row: mint check on the left, title + "You/Maya · 9:12 am"
/// subtitle in the middle, member-colored initial avatar on the right.
struct GraveyardRow: View {
    let chore: HearthChore
    let currentUserID: String?
    let members: [HearthMember]
    var showsBottomDivider: Bool = true

    private var isMine: Bool {
        guard let me = currentUserID, let by = chore.completedBy else { return false }
        return by == me
    }

    private var completerName: String {
        if isMine { return "You" }
        if let by = chore.completedBy,
           let name = MembersStore.displayName(for: by, in: members) {
            return name
        }
        return "Someone"
    }

    private var completerInitial: String {
        if isMine {
            return MembersStore.initial(for: currentUserID ?? "", in: members)
                ?? "V"
        }
        if let by = chore.completedBy, let initial = MembersStore.initial(for: by, in: members) {
            return initial
        }
        return "?"
    }

    private var timeString: String {
        guard let completedAt = chore.completedAt,
              let date = parseCompletedAt(completedAt) else { return "" }
        return Self.formatter.string(from: date).lowercased()
    }

    private static let formatter: DateFormatter = {
        let f = DateFormatter()
        f.dateFormat = "h:mm a"
        return f
    }()

    var body: some View {
        VStack(spacing: 0) {
            HStack(spacing: 14) {
                checkmark
                VStack(alignment: .leading, spacing: 3) {
                    Text(chore.title)
                        .font(.hearthBody(size: 16, weight: .medium))
                        .foregroundStyle(Color.hearthTextPrimary)
                        .lineLimit(1)
                    Text("\(completerName) · \(timeString)")
                        .font(.hearthBody(size: 13))
                        .foregroundStyle(Color.hearthTextSecondary)
                }
                Spacer(minLength: 8)
                MemberAvatar(
                    initial: completerInitial,
                    isSelf: isMine,
                    size: 26,
                    borderColor: .clear
                )
            }
            .frame(minHeight: 60)

            if showsBottomDivider {
                Rectangle()
                    .fill(Color.hearthDoneDivider)
                    .frame(height: 1)
            }
        }
    }

    private var checkmark: some View {
        ZStack {
            Circle().fill(Color.hearthMint)
            Image(systemName: "checkmark")
                .font(.system(size: 11, weight: .bold))
                .foregroundStyle(Color.hearthMintInk)
        }
        .frame(width: 26, height: 26)
    }

    private func parseCompletedAt(_ s: String) -> Date? {
        ISO8601DateFormatter.hearthISO().date(from: s) ?? ISO8601DateFormatter().date(from: s)
    }
}

extension Color {
    /// Hairline divider between rows inside a Done section card.
    static let hearthDoneDivider = Color(
        light: Color(red: 0xEF / 255, green: 0xED / 255, blue: 0xF4 / 255),
        dark: Color(red: 0x3A / 255, green: 0x37 / 255, blue: 0x44 / 255)
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
    return VStack(spacing: 0) {
        GraveyardRow(
            chore: HearthChore(
                id: "1", listId: "l", creatorId: "me",
                title: "Unload the dishwasher", notes: nil, state: "completed",
                claimedBy: "u_maya", claimedAt: nil, willCompleteBy: nil,
                completedBy: "u_maya",
                completedAt: "2026-10-27T16:12:00.000Z",
                createdAt: "", updatedAt: "", overdue: false
            ),
            currentUserID: "me", members: members
        )
        GraveyardRow(
            chore: HearthChore(
                id: "2", listId: "l", creatorId: "me",
                title: "Feed the cat", notes: nil, state: "completed",
                claimedBy: "me", claimedAt: nil, willCompleteBy: nil,
                completedBy: "me",
                completedAt: "2026-10-27T15:03:00.000Z",
                createdAt: "", updatedAt: "", overdue: false
            ),
            currentUserID: "me", members: members,
            showsBottomDivider: false
        )
    }
    .padding(.horizontal, 16)
    .background(Color.hearthSurface)
    .padding(20)
    .background(Color.hearthBackground)
}
#endif
