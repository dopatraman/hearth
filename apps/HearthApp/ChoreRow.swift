import SwiftUI

/// Row representation of a single `HearthChore` in a List. Visual-only; takes
/// no actions. Expands in Slice 5 with claim/complete swipe affordances and
/// claimed-by chips, so it earns its own file now rather than waiting.
struct ChoreRow: View {
    let chore: HearthChore

    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(chore.title)
                .font(.body)
                .foregroundStyle(Color.hearthTextPrimary)
            if let notes = chore.notes, !notes.isEmpty {
                Text(notes)
                    .font(.footnote)
                    .foregroundStyle(Color.hearthTextSecondary)
                    .lineLimit(2)
            }
        }
        .padding(.vertical, 6)
    }
}

#Preview {
    List {
        ChoreRow(chore: HearthChore(
            id: "1",
            listId: "l",
            creatorId: "u",
            title: "Take out trash",
            notes: "Both bins; recycling truck comes Tuesday",
            state: "open",
            claimedBy: nil, claimedAt: nil,
            willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "", updatedAt: ""
        ))
        ChoreRow(chore: HearthChore(
            id: "2",
            listId: "l",
            creatorId: "u",
            title: "Water plants",
            notes: nil,
            state: "open",
            claimedBy: nil, claimedAt: nil,
            willCompleteBy: nil,
            completedBy: nil, completedAt: nil,
            createdAt: "", updatedAt: ""
        ))
    }
    .listStyle(.plain)
}
