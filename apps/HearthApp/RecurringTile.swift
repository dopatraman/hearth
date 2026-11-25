import SwiftUI

/// A single circular icon tile in the Recurring rail. Pure view, no state.
/// The surrounding rail owns the "leaving" and "arriving" animations and
/// toggles them through `phase`.
struct RecurringTile: View {
    enum Phase: Equatable {
        case idle
        case leaving
        case arriving
    }

    let title: String
    let iconSymbol: String
    let phase: Phase
    let onTap: () -> Void

    init(
        title: String,
        iconSymbol: String = "arrow.triangle.2.circlepath",
        phase: Phase = .idle,
        onTap: @escaping () -> Void
    ) {
        self.title = title
        self.iconSymbol = iconSymbol
        self.phase = phase
        self.onTap = onTap
    }

    var body: some View {
        Button(action: onTap) {
            VStack(spacing: 8) {
                ZStack {
                    Circle()
                        .fill(Color.hearthRecurringTileBubble)
                    Image(systemName: iconSymbol)
                        .font(.system(size: 22, weight: .medium))
                        .foregroundStyle(Color.hearthRecurringAccent)
                        .accessibilityHidden(true)
                }
                .frame(width: 56, height: 56)

                Text(title)
                    .font(.hearthBody(size: 12, weight: .medium))
                    .foregroundStyle(Color.hearthTextPrimary)
                    .lineLimit(2)
                    .multilineTextAlignment(.center)
                    .frame(height: 30)
            }
            .frame(width: 76)
            .contentShape(Rectangle())
        }
        .buttonStyle(.plain)
        .accessibilityLabel("Claim \(title)")
        .scaleEffect(scale)
        .offset(y: offsetY)
        .opacity(opacity)
    }

    private var scale: CGFloat {
        switch phase {
        case .idle: return 1
        case .leaving: return 0.5
        case .arriving: return 1
        }
    }

    private var offsetY: CGFloat {
        switch phase {
        case .idle: return 0
        case .leaving: return 34
        case .arriving: return 0
        }
    }

    private var opacity: Double {
        switch phase {
        case .idle, .arriving: return 1
        case .leaving: return 0
        }
    }
}

#if DEBUG
#Preview {
    HStack(spacing: 4) {
        RecurringTile(title: "Make the bed", iconSymbol: "bed.double") {}
        RecurringTile(title: "Unload the dishwasher", iconSymbol: "sparkles") {}
        RecurringTile(title: "Water the plants", iconSymbol: "leaf") {}
    }
    .padding(16)
    .background(Color.hearthBackground)
}
#endif
