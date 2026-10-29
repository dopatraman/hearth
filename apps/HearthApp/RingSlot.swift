import SwiftUI

/// Left-side 44pt hit target on a chore row. Visual depends on the chore's
/// role relative to the current user:
/// - **mine** → 26pt lavender-outlined empty circle (completion affordance)
/// - **open** → 26pt dashed gray ring
/// - **other** → 28pt sky circle with the member's initial
struct RingSlot: View {
    let role: ChoreRole
    let otherInitial: String?
    var isCompleting: Bool = false

    var body: some View {
        ZStack {
            switch role {
            case .mine:
                ZStack {
                    Circle()
                        .stroke(Color.hearthLavender, lineWidth: 2)
                        .frame(width: 26, height: 26)
                        .opacity(isCompleting ? 0 : 1)
                        .scaleEffect(isCompleting ? 0.85 : 1)
                    ZStack {
                        Circle().fill(Color.hearthMint)
                        Image(systemName: "checkmark")
                            .font(.system(size: 12, weight: .bold))
                            .foregroundStyle(Color.hearthMintInk)
                    }
                    .frame(width: 26, height: 26)
                    .opacity(isCompleting ? 1 : 0)
                    .scaleEffect(isCompleting ? 1 : 0.6)
                }
                .animation(.spring(response: 0.3, dampingFraction: 0.72), value: isCompleting)
            case .open:
                Circle()
                    .strokeBorder(
                        Color.hearthRingOpen,
                        style: StrokeStyle(lineWidth: 2, dash: [3, 2.5])
                    )
                    .frame(width: 26, height: 26)
            case .other:
                MemberAvatar(
                    initial: otherInitial ?? "?",
                    isSelf: false,
                    size: 28,
                    borderColor: .clear
                )
            }
        }
        .frame(width: 44, height: 44)
    }
}

extension Color {
    /// Dashed-ring color for the "open" (unclaimed) state.
    static let hearthRingOpen = Color(
        light: Color(red: 0xB7 / 255, green: 0xB3 / 255, blue: 0xC6 / 255),
        dark: Color(red: 0x6E / 255, green: 0x6A / 255, blue: 0x80 / 255)
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
    HStack(spacing: 20) {
        RingSlot(role: .mine, otherInitial: nil)
        RingSlot(role: .open, otherInitial: nil)
        RingSlot(role: .other, otherInitial: "M")
    }
    .padding()
    .background(Color.hearthBackground)
}
#endif
