import SwiftUI

/// The top of the Backlog screen: date, "Backlog" display title, summary line,
/// and the avatar stack + settings gear in the trailing position.
struct BacklogHeader: View {
    let chores: [HearthChore]
    let members: [HearthMember]
    let currentUserID: String?
    let now: Date
    let onSettings: () -> Void
    let selectedMemberID: String?
    let onTapMember: ((HearthMember) -> Void)?

    init(
        chores: [HearthChore],
        members: [HearthMember],
        currentUserID: String?,
        now: Date = Date(),
        onSettings: @escaping () -> Void,
        selectedMemberID: String? = nil,
        onTapMember: ((HearthMember) -> Void)? = nil
    ) {
        self.chores = chores
        self.members = members
        self.currentUserID = currentUserID
        self.now = now
        self.onSettings = onSettings
        self.selectedMemberID = selectedMemberID
        self.onTapMember = onTapMember
    }

    private var dateLine: String {
        now.formatted(.dateTime.weekday(.wide).month(.wide).day())
    }

    var body: some View {
        VStack(alignment: .leading, spacing: 12) {
            HStack(alignment: .top, spacing: 12) {
                VStack(alignment: .leading, spacing: 0) {
                    Text(dateLine)
                        .font(.hearthBody(size: 13, weight: .medium))
                        .foregroundStyle(Color.hearthTextSecondary)
                        .padding(.bottom, 4)

                    Text("Backlog")
                        .font(.hearthDisplay(size: 36, weight: .bold))
                        .foregroundStyle(Color.hearthTextPrimary)
                }

                Spacer(minLength: 0)

                HStack(spacing: 6) {
                    AvatarStack(
                        members: members,
                        currentUserID: currentUserID,
                        selectedMemberID: selectedMemberID,
                        onTapMember: onTapMember
                    )

                    Button(action: onSettings) {
                        Image(systemName: "gearshape")
                            .font(.system(size: 20, weight: .regular))
                            .foregroundStyle(Color.hearthTextSecondary)
                            .frame(width: 44, height: 44)
                            .glassEffect(.regular.interactive(), in: Circle())
                    }
                    .buttonStyle(.plain)
                    .accessibilityLabel("Settings")
                }
                .padding(.top, 6)
            }

            Text(summaryLine(chores: chores, currentUserID: currentUserID))
                .font(.hearthBody(size: 16))
                .foregroundStyle(Color.hearthTextSecondary)
                .fixedSize(horizontal: false, vertical: true)
                .frame(maxWidth: .infinity, alignment: .leading)
        }
        .padding(.horizontal, 20)
        .padding(.top, 8)
        .padding(.bottom, 12)
    }
}

#if DEBUG
#Preview("Three mine, two late") {
    let me = HearthUser(id: "u_self", displayName: "V", notificationsEnabled: true)
    let members = [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]
    let chores: [HearthChore] = [
        HearthChore(id: "a", listId: "l", creatorId: me.id, title: "x",
                    notes: nil, state: "open",
                    claimedBy: me.id, claimedAt: nil, willCompleteBy: nil,
                    completedBy: nil, completedAt: nil,
                    createdAt: "", updatedAt: "", overdue: true),
        HearthChore(id: "b", listId: "l", creatorId: me.id, title: "x",
                    notes: nil, state: "open",
                    claimedBy: me.id, claimedAt: nil, willCompleteBy: nil,
                    completedBy: nil, completedAt: nil,
                    createdAt: "", updatedAt: "", overdue: true),
        HearthChore(id: "c", listId: "l", creatorId: me.id, title: "x",
                    notes: nil, state: "open",
                    claimedBy: me.id, claimedAt: nil, willCompleteBy: nil,
                    completedBy: nil, completedAt: nil,
                    createdAt: "", updatedAt: "", overdue: false),
    ]
    return BacklogHeader(
        chores: chores,
        members: members,
        currentUserID: me.id,
        onSettings: {}
    )
    .background(Color.hearthBackground)
}

#Preview("Nothing on your plate") {
    let members = [
        HearthMember(id: "u_self", displayName: "Venkat"),
        HearthMember(id: "u_maya", displayName: "Maya"),
    ]
    return BacklogHeader(
        chores: [],
        members: members,
        currentUserID: "u_self",
        onSettings: {}
    )
    .background(Color.hearthBackground)
}
#endif
