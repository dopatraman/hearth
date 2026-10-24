import SwiftUI

/// Row representation of a single `HearthChore`. Carries visual state based
/// on claim status and offers the tap-to-complete affordance (mint check)
/// when the current user is the claimant.
struct ChoreRow: View {
    let chore: HearthChore
    /// The current user's id, used to decide whether a chore is "mine" or
    /// "someone else's". Nil before auth bootstrap finishes.
    let currentUserID: String?
    /// Called when the user taps the mint check button on their own chore.
    let onComplete: () -> Void

    private var mine: Bool {
        guard let currentUserID else { return false }
        return chore.claimedBy == currentUserID
    }

    private var claimedByOther: Bool {
        guard let claimant = chore.claimedBy else { return false }
        return claimant != currentUserID
    }

    private var accentColor: Color {
        if chore.overdue { return .hearthPeach }
        if mine { return .hearthLavender }
        return .clear
    }

    var body: some View {
        HStack(spacing: 12) {
            // Left accent — peach if overdue, lavender if mine, invisible otherwise.
            Rectangle()
                .fill(accentColor)
                .frame(width: 3)
                .cornerRadius(1.5)

            VStack(alignment: .leading, spacing: 4) {
                HStack(spacing: 6) {
                    Text(chore.title)
                        .font(.body)
                        .foregroundStyle(Color.hearthTextPrimary)
                    if chore.overdue {
                        Text("OVERDUE")
                            .font(.caption2.weight(.bold))
                            .foregroundStyle(.white)
                            .padding(.horizontal, 6)
                            .padding(.vertical, 2)
                            .background(Color.hearthPeach, in: Capsule())
                    }
                }
                if let notes = chore.notes, !notes.isEmpty {
                    Text(notes)
                        .font(.footnote)
                        .foregroundStyle(Color.hearthTextSecondary)
                        .lineLimit(2)
                }
                if let wcb = chore.willCompleteBy,
                   let date = ISO8601DateFormatter.hearthISO().date(from: wcb) {
                    Text("by \(date.formatted(date: .abbreviated, time: .shortened))")
                        .font(.caption)
                        .foregroundStyle(chore.overdue ? Color.hearthPeach : Color.hearthTextSecondary)
                }
            }

            Spacer()

            if claimedByOther {
                // Sky chip indicating someone else owns this.
                Image(systemName: "person.fill")
                    .font(.footnote)
                    .foregroundStyle(.white)
                    .padding(8)
                    .background(Color.hearthSky, in: Circle())
            } else if mine {
                // Mint check — tap to complete.
                Button(action: onComplete) {
                    Image(systemName: "checkmark")
                        .font(.system(size: 14, weight: .semibold))
                        .foregroundStyle(.white)
                        .frame(width: 32, height: 32)
                        .background(Color.hearthMint, in: Circle())
                }
                .buttonStyle(.plain)
                .sensoryFeedback(.success, trigger: chore.id)
            }
        }
        .padding(.vertical, 6)
    }
}

#Preview {
    List {
        ChoreRow(
            chore: previewChore(title: "Take out trash", notes: "Both bins", claimedBy: nil),
            currentUserID: "me",
            onComplete: {}
        )
        ChoreRow(
            chore: previewChore(title: "Water plants", notes: nil, claimedBy: "me"),
            currentUserID: "me",
            onComplete: {}
        )
        ChoreRow(
            chore: previewChore(title: "Mop floor", notes: nil, claimedBy: "someone-else"),
            currentUserID: "me",
            onComplete: {}
        )
    }
    .listStyle(.plain)
}

private func previewChore(title: String, notes: String?, claimedBy: String?) -> HearthChore {
    HearthChore(
        id: UUID().uuidString,
        listId: "l",
        creatorId: "me",
        title: title,
        notes: notes,
        state: "open",
        claimedBy: claimedBy,
        claimedAt: nil,
        willCompleteBy: nil,
        completedBy: nil,
        completedAt: nil,
        createdAt: "", updatedAt: "",
        overdue: false
    )
}
