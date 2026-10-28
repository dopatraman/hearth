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
/// self first (lavender), others after (sky), overlapping. Each avatar is a
/// filter affordance: tap toggles the backlog filter to show that member's
/// chores. When any member is selected, the others fade; the selected one
/// gains a lavender-ink ring and scales up slightly.
struct AvatarStack: View {
    let members: [HearthMember]
    let currentUserID: String?
    var size: CGFloat = 44
    var selectedMemberID: String?
    var onTapMember: ((HearthMember) -> Void)?

    var body: some View {
        HStack(spacing: -12) {
            ForEach(ordered) { m in
                avatarButton(for: m)
            }
        }
        .accessibilityElement(children: .contain)
        .accessibilityLabel("Household")
    }

    @ViewBuilder
    private func avatarButton(for m: HearthMember) -> some View {
        let selected = selectedMemberID == m.id
        let dimmed = selectedMemberID != nil && !selected
        let avatar = MemberAvatar(
            initial: MembersStore.initial(for: m.id, in: members) ?? "?",
            isSelf: m.id == currentUserID,
            size: size,
            borderColor: selected ? Color.hearthLavenderInk : Color.hearthBackground
        )
        .overlay(
            Circle()
                .stroke(Color.hearthLavenderInk, lineWidth: selected ? 3 : 0)
        )
        .scaleEffect(selected ? 1.06 : 1)
        .opacity(dimmed ? 0.45 : 1)
        .zIndex(selected ? 1 : 0)
        .animation(.spring(response: 0.35, dampingFraction: 0.8), value: selectedMemberID)

        if let onTapMember {
            Button {
                onTapMember(m)
            } label: {
                avatar.contentShape(Circle())
            }
            .buttonStyle(.plain)
            .accessibilityLabel(m.displayName ?? "member")
            .accessibilityValue(selected ? "Filter on" : "Filter off")
        } else {
            avatar
        }
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
        currentUserID: "u_self",
        selectedMemberID: "u_self",
        onTapMember: { _ in }
    )
    .padding()
    .background(Color.hearthBackground)
}
