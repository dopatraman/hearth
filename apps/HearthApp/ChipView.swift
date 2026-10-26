import SwiftUI

/// Capsule chip rendered beneath a chore title to surface time pressure.
struct ChipView: View {
    let chip: Chip

    var body: some View {
        Text(chip.text)
            .font(.hearthBody(size: 12, weight: .semibold))
            .foregroundStyle(ink)
            .padding(.horizontal, 9)
            .padding(.vertical, 3)
            .background(bg, in: Capsule())
    }

    private var bg: Color {
        switch chip.kind {
        case .overdue: return .hearthChipOverdueBg
        case .tonight: return .hearthChipTonightBg
        case .scheduled: return .hearthChipScheduledBg
        }
    }

    private var ink: Color {
        switch chip.kind {
        case .overdue: return .hearthChipOverdueInk
        case .tonight: return .hearthChipTonightInk
        case .scheduled: return .hearthChipScheduledInk
        }
    }
}

extension Color {
    // Overdue — peach
    static let hearthChipOverdueBg = Color(
        light: Color(red: 0xFF / 255, green: 0xE2 / 255, blue: 0xD1 / 255),
        dark: Color(red: 0x6B / 255, green: 0x3D / 255, blue: 0x24 / 255)
    )
    static let hearthChipOverdueInk = Color(
        light: Color(red: 0x9A / 255, green: 0x43 / 255, blue: 0x17 / 255),
        dark: Color(red: 0xF4 / 255, green: 0xC4 / 255, blue: 0xA3 / 255)
    )

    // Tonight — amber
    static let hearthChipTonightBg = Color(
        light: Color(red: 0xFF / 255, green: 0xF1 / 255, blue: 0xBF / 255),
        dark: Color(red: 0x5A / 255, green: 0x45 / 255, blue: 0x15 / 255)
    )
    static let hearthChipTonightInk = Color(
        light: Color(red: 0x7A / 255, green: 0x5A / 255, blue: 0x00 / 255),
        dark: Color(red: 0xFF / 255, green: 0xDE / 255, blue: 0x8A / 255)
    )

    // Scheduled — muted lavender/gray
    static let hearthChipScheduledBg = Color(
        light: Color(red: 0xEF / 255, green: 0xED / 255, blue: 0xF4 / 255),
        dark: Color(red: 0x3A / 255, green: 0x37 / 255, blue: 0x44 / 255)
    )
    static let hearthChipScheduledInk = Color(
        light: Color(red: 0x5E / 255, green: 0x5A / 255, blue: 0x6E / 255),
        dark: Color(red: 0xBE / 255, green: 0xB9 / 255, blue: 0xCC / 255)
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
    VStack(alignment: .leading, spacing: 10) {
        ChipView(chip: Chip(text: "2 days late", kind: .overdue))
        ChipView(chip: Chip(text: "Tonight", kind: .tonight))
        ChipView(chip: Chip(text: "by Fri", kind: .scheduled))
        ChipView(chip: Chip(text: "by Oct 30", kind: .scheduled))
    }
    .padding()
    .background(Color.hearthBackground)
}
#endif
