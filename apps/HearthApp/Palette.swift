import SwiftUI

/// Hearth palette — semantic names, light + dark values. Hex values match the
/// design canvas (Main.dc.html). A color here always carries meaning: a
/// lavender highlight ALWAYS means "mine"; a peach badge ALWAYS means
/// "overdue". Users learn the system by using it.
extension Color {
    /// Cool lavender-gray ground in light, warm-cool near-black in dark.
    static let hearthBackground = Color(
        light: Color(red: 0xF4 / 255, green: 0xF3 / 255, blue: 0xF8 / 255),
        dark: Color(red: 0x1C / 255, green: 0x1A / 255, blue: 0x1F / 255)
    )

    /// Primary text.
    static let hearthTextPrimary = Color(
        light: Color(red: 0x1D / 255, green: 0x1B / 255, blue: 0x26 / 255),
        dark: Color(red: 0xF0 / 255, green: 0xEC / 255, blue: 0xF2 / 255)
    )

    /// Metadata, timestamps, hints.
    static let hearthTextSecondary = Color(
        light: Color(red: 0x6E / 255, green: 0x6A / 255, blue: 0x80 / 255),
        dark: Color(red: 0xA6 / 255, green: 0xA1 / 255, blue: 0xB2 / 255)
    )

    /// Avatar background for the current user. Lighter pastel than the ring
    /// stroke — used as a surface fill, not as an edge.
    static let hearthLavender = Color(
        light: Color(red: 0xE6 / 255, green: 0xDE / 255, blue: 0xFB / 255),
        dark: Color(red: 0x4A / 255, green: 0x3F / 255, blue: 0x6B / 255)
    )

    /// Deeper lavender used for the "mine" ring stroke. The empty-circle
    /// completion affordance on a mine-chore card is drawn in this color.
    static let hearthRingMineStroke = Color(
        light: Color(red: 0x8F / 255, green: 0x78 / 255, blue: 0xE0 / 255),
        dark: Color(red: 0xB4 / 255, green: 0xA2 / 255, blue: 0xEE / 255)
    )

    /// Mid-saturation lavender used for the "Claimed" section dot.
    static let hearthLavenderDot = Color(
        light: Color(red: 0xB9 / 255, green: 0xA6 / 255, blue: 0xF2 / 255),
        dark: Color(red: 0x8A / 255, green: 0x7B / 255, blue: 0xC8 / 255)
    )

    /// Success / completed — the mint check and graveyard accents.
    static let hearthMint = Color(
        light: Color(red: 0xB6 / 255, green: 0xE4 / 255, blue: 0xCF / 255),
        dark: Color(red: 0x7A / 255, green: 0xA8 / 255, blue: 0x96 / 255)
    )

    /// Mint check path drawn inside the mine ring. Darker than the fill.
    static let hearthMintInk = Color(
        light: Color(red: 0x1F / 255, green: 0x6B / 255, blue: 0x4A / 255),
        dark: Color(red: 0xB6 / 255, green: 0xE4 / 255, blue: 0xCF / 255)
    )

    /// Warning / overdue — chores past their will_complete_by (also chip ink).
    static let hearthPeach = Color(
        light: Color(red: 0xF4 / 255, green: 0xC4 / 255, blue: 0xA3 / 255),
        dark: Color(red: 0xB8 / 255, green: 0x8A / 255, blue: 0x6E / 255)
    )

    /// Avatar background for other household members. Lighter pastel than
    /// the saturated sky formerly used.
    static let hearthSky = Color(
        light: Color(red: 0xD6 / 255, green: 0xE8 / 255, blue: 0xFB / 255),
        dark: Color(red: 0x3F / 255, green: 0x4F / 255, blue: 0x6B / 255)
    )

    /// Ink color for text/initial drawn on `hearthLavender` surfaces (self avatar).
    static let hearthLavenderInk = Color(
        light: Color(red: 0x4B / 255, green: 0x3A / 255, blue: 0x8C / 255),
        dark: Color(red: 0xC7 / 255, green: 0xB9 / 255, blue: 0xEC / 255)
    )

    /// Ink color for text/initial drawn on `hearthSky` surfaces (other-member avatar).
    static let hearthSkyInk = Color(
        light: Color(red: 0x1F / 255, green: 0x5A / 255, blue: 0x8F / 255),
        dark: Color(red: 0xB4 / 255, green: 0xD1 / 255, blue: 0xEF / 255)
    )

    /// Card/sheet surface. White in light, warm near-black in dark. Used by
    /// chore row cards and the New Chore composer.
    static let hearthSurface = Color(
        light: .white,
        dark: Color(red: 0x2A / 255, green: 0x27 / 255, blue: 0x24 / 255)
    )

    /// Muted chevron / small-icon tint used inside rows.
    static let hearthChevron = Color(
        light: Color(red: 0x9E / 255, green: 0x9A / 255, blue: 0xAF / 255),
        dark: Color(red: 0x7E / 255, green: 0x7A / 255, blue: 0x8F / 255)
    )

    /// Soft inset button background that reads against a `hearthSurface` card
    /// in both themes — used for action pills, chip-rest state, and the X-close.
    static let hearthInsetFill = Color(
        light: Color(red: 0xF4 / 255, green: 0xF3 / 255, blue: 0xF8 / 255),
        dark: Color(red: 0x3A / 255, green: 0x37 / 255, blue: 0x44 / 255)
    )

    /// Ink color for text drawn on `hearthInsetFill` buttons.
    static let hearthInsetInk = Color(
        light: Color(red: 0x3A / 255, green: 0x36 / 255, blue: 0x48 / 255),
        dark: Color(red: 0xC9 / 255, green: 0xC4 / 255, blue: 0xD6 / 255)
    )

    /// Switch track background in the off state (New Chore toggle).
    static let hearthSwitchTrackOff = Color(
        light: Color(red: 0xDC / 255, green: 0xD9 / 255, blue: 0xE5 / 255),
        dark: Color(red: 0x4A / 255, green: 0x47 / 255, blue: 0x54 / 255)
    )
}

private extension Color {
    init(light: Color, dark: Color) {
        self.init(uiColor: UIColor { trait in
            trait.userInterfaceStyle == .dark
                ? UIColor(dark)
                : UIColor(light)
        })
    }
}
