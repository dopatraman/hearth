import SwiftUI
import UIKit

/// Returns the index of `selected` in `options`. Falls back to 0 if the
/// selection isn't in the list — a sensible default so the sliding indicator
/// stays rendered rather than disappearing.
func segmentIndex<Option: Hashable>(selected: Option, options: [Option]) -> Int {
    options.firstIndex(of: selected) ?? 0
}

/// Pill-shaped 2-segment control with a white sliding indicator. Replaces the
/// system `Picker(.segmented)` on the Backlog filter row. Spring curve
/// perceptually matches the design canvas's cubic-bezier(.3, 1.35, .5, 1).
struct SegmentedPill<Option: Hashable>: View {
    @Binding var selection: Option
    let options: [(Option, String)]
    var segmentWidth: CGFloat = 88
    var segmentHeight: CGFloat = 32

    @Namespace private var indicator

    var body: some View {
        HStack(spacing: 0) {
            ForEach(Array(options.enumerated()), id: \.offset) { _, pair in
                segmentButton(for: pair.0, label: pair.1)
            }
        }
        .padding(4)
        .background(Color.hearthSegmentTrack, in: Capsule())
        .animation(.spring(response: 0.35, dampingFraction: 0.78), value: selection)
    }

    private func segmentButton(for option: Option, label: String) -> some View {
        let isSelected = selection == option
        return Button {
            guard selection != option else { return }
            UISelectionFeedbackGenerator().selectionChanged()
            selection = option
        } label: {
            ZStack {
                if isSelected {
                    Capsule()
                        .fill(Color.hearthSegmentIndicator)
                        .shadow(color: .black.opacity(0.08), radius: 2, y: 1)
                        .shadow(color: .black.opacity(0.06), radius: 10, y: 4)
                        .matchedGeometryEffect(id: "pill-indicator", in: indicator)
                }
                Text(label)
                    .font(.hearthBody(size: 14, weight: .semibold))
                    .foregroundStyle(isSelected ? Color.hearthTextPrimary : Color.hearthTextSecondary)
            }
            .frame(width: segmentWidth, height: segmentHeight)
            .contentShape(Capsule())
        }
        .buttonStyle(.plain)
        .accessibilityLabel(label)
        .accessibilityAddTraits(isSelected ? [.isButton, .isSelected] : [.isButton])
    }
}

extension Color {
    /// Muted track behind the sliding indicator in `SegmentedPill`.
    static let hearthSegmentTrack = Color(
        light: Color(red: 0xE9 / 255, green: 0xE7 / 255, blue: 0xF0 / 255),
        dark: Color(red: 0x33 / 255, green: 0x30 / 255, blue: 0x3B / 255)
    )

    /// The sliding white indicator in `SegmentedPill`.
    static let hearthSegmentIndicator = Color(
        light: .white,
        dark: Color(red: 0x4A / 255, green: 0x47 / 255, blue: 0x52 / 255)
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
#Preview("All selected") {
    @Previewable @State var selection: BacklogView.Filter = .all
    return SegmentedPill(
        selection: $selection,
        options: [(BacklogView.Filter.all, "All"), (BacklogView.Filter.mine, "Mine")]
    )
    .padding()
    .background(Color.hearthBackground)
}
#endif
