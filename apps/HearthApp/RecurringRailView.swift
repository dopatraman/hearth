import SwiftUI
import UIKit

/// Horizontal rail of `RecurringTile`s pinned at the top of the Backlog.
/// Tapping a tile animates it out (shrink + drop) then forwards to `onClaim`.
/// When the store flags a tile as recently replenished, the view plays the
/// "back" animation for that tile. The rail itself owns no server state.
struct RecurringRailView: View {
    let items: [HearthChore]
    let recentlyReplenished: Set<String>
    let onClaim: (HearthChore) -> Void

    @State private var leavingIDs: Set<String> = []

    var body: some View {
        ScrollView(.horizontal, showsIndicators: false) {
            HStack(spacing: 4) {
                ForEach(items) { chore in
                    RecurringTile(
                        title: chore.title,
                        iconSymbol: chore.icon ?? "arrow.triangle.2.circlepath",
                        phase: phase(for: chore.id)
                    ) {
                        beginClaim(chore)
                    }
                    .transition(.asymmetric(
                        insertion: .scale(scale: 0.4).combined(with: .opacity).combined(with: .offset(y: 30)),
                        removal: .opacity
                    ))
                }
            }
            .padding(.horizontal, 14)
            .padding(.vertical, 4)
        }
        .animation(.spring(response: 0.6, dampingFraction: 0.6), value: items.map(\.id))
    }

    private func phase(for id: String) -> RecurringTile.Phase {
        if leavingIDs.contains(id) { return .leaving }
        if recentlyReplenished.contains(id) { return .arriving }
        return .idle
    }

    private func beginClaim(_ chore: HearthChore) {
        guard !leavingIDs.contains(chore.id) else { return }
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        withAnimation(.easeIn(duration: 0.38)) {
            leavingIDs.insert(chore.id)
        }
        Task {
            try? await Task.sleep(for: .milliseconds(360))
            onClaim(chore)
            leavingIDs.remove(chore.id)
        }
    }
}

#if DEBUG
#Preview {
    let items: [HearthChore] = [
        previewRecurringChore(id: "r1", title: "Make the bed", icon: "bed.double"),
        previewRecurringChore(id: "r2", title: "Unload the dishwasher", icon: "sparkles"),
        previewRecurringChore(id: "r3", title: "Water the plants", icon: "leaf"),
        previewRecurringChore(id: "r4", title: "Wipe the counters", icon: "sparkle"),
        previewRecurringChore(id: "r5", title: "Take out the trash", icon: "trash"),
    ]
    return VStack(alignment: .leading, spacing: 12) {
        HStack(spacing: 8) {
            Image(systemName: "arrow.triangle.2.circlepath")
                .foregroundStyle(Color.hearthRecurringAccent)
            Text("Recurring")
                .font(.hearthBody(size: 14, weight: .semibold))
        }
        .padding(.leading, 22)
        RecurringRailView(items: items, recentlyReplenished: []) { _ in }
    }
    .padding(.vertical, 20)
    .background(Color.hearthBackground)
}

private func previewRecurringChore(id: String, title: String, icon: String) -> HearthChore {
    HearthChore(
        id: id, listId: "l", creatorId: "me", title: title,
        notes: nil, state: "open",
        claimedBy: nil, claimedAt: nil, willCompleteBy: nil,
        completedBy: nil, completedAt: nil,
        createdAt: "2026-10-20T10:00:00.000Z",
        updatedAt: "2026-10-20T10:00:00.000Z",
        overdue: false,
        isRecurring: true,
        icon: icon
    )
}
#endif
