import SwiftUI

/// Row representation of a completed chore in the Graveyard. Mint check +
/// title + completed_at. Visual-only. Will grow when we show the completer's
/// avatar or the time-to-complete, so earns its own file now.
struct GraveyardRow: View {
    let chore: HearthChore

    var body: some View {
        HStack(spacing: 12) {
            Image(systemName: "checkmark.circle.fill")
                .foregroundStyle(Color.hearthMint)
            VStack(alignment: .leading, spacing: 2) {
                Text(chore.title)
                    .font(.body)
                    .foregroundStyle(Color.hearthTextPrimary)
                if let completedAt = chore.completedAt {
                    Text(completedAt)
                        .font(.caption)
                        .foregroundStyle(Color.hearthTextSecondary)
                }
            }
        }
    }
}

#Preview {
    List {
        GraveyardRow(chore: HearthChore(
            id: "1", listId: "l", creatorId: "u",
            title: "Take out trash",
            notes: nil,
            state: "completed",
            claimedBy: "u", claimedAt: nil,
            willCompleteBy: nil,
            completedBy: "u",
            completedAt: "2026-10-23T19:30:00.000Z",
            createdAt: "", updatedAt: "",
            overdue: false
        ))
    }
    .listStyle(.plain)
}
