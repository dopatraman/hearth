import SwiftUI

struct MemberAvatar: View {
    let initial: String
    let isSelf: Bool
    var size: CGFloat = 32
    var borderColor: Color = .hearthBackground

    var body: some View {
        Circle()
            .fill(isSelf ? Color.hearthLavender : Color.hearthSky)
            .frame(width: size, height: size)
            .overlay(
                Text(initial)
                    .font(.system(size: size * 0.42, weight: .semibold, design: .rounded))
                    .foregroundStyle(isSelf ? Color.hearthLavenderInk : Color.hearthSkyInk)
            )
            .overlay(
                Circle().stroke(borderColor, lineWidth: 2)
            )
    }
}

/// Stacked circular avatars for the current household, in a fixed order:
/// self first (lavender), others after (sky), overlapping by 10pt.
struct AvatarStack: View {
    let members: [HearthMember]
    let currentUserID: String?

    var body: some View {
        HStack(spacing: -10) {
            ForEach(ordered) { m in
                MemberAvatar(
                    initial: MembersStore.initial(for: m.id, in: members) ?? "?",
                    isSelf: m.id == currentUserID
                )
            }
        }
        .accessibilityElement(children: .combine)
        .accessibilityLabel("Household")
    }

    private var ordered: [HearthMember] {
        members.sorted { a, b in
            let aIsSelf = a.id == currentUserID
            let bIsSelf = b.id == currentUserID
            if aIsSelf != bIsSelf { return aIsSelf }
            return (a.displayName ?? a.id) < (b.displayName ?? b.id)
        }
    }
}

#Preview {
    AvatarStack(
        members: [
            HearthMember(id: "u_self", displayName: "Venkat"),
            HearthMember(id: "u_maya", displayName: "Maya"),
        ],
        currentUserID: "u_self"
    )
    .padding()
    .background(Color.hearthBackground)
}
