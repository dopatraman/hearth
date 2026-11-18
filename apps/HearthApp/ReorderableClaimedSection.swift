import SwiftUI
import UIKit

/// Hand-rolled long-press + drag reorder for the Backlog's Claimed section.
/// Owns all the drag state locally; the parent only sees lift transitions,
/// first-use of the gesture, and the async commit when the user drops.
struct ReorderableClaimedSection<Row: View>: View {
    let items: [HearthChore]
    @Binding var hasUsedReorder: Bool
    let onLiftChange: (Bool) -> Void
    let onReorder: ([String]) async -> Void
    @ViewBuilder let rowContent: (HearthChore) -> Row

    @State private var liftedID: String?
    @State private var pressedID: String?
    @State private var dragOffset: CGFloat = 0
    @State private var dragFromIndex: Int?
    @State private var dragToIndex: Int?

    private static var step: CGFloat { 76 }

    var body: some View {
        VStack(spacing: 10) {
            ForEach(Array(items.enumerated()), id: \.element.id) { index, chore in
                row(chore: chore, index: index)
            }
        }
    }

    private func row(chore: HearthChore, index: Int) -> some View {
        let isLifted = liftedID == chore.id
        let isPressing = pressedID == chore.id
        let yShift: CGFloat = {
            guard let from = dragFromIndex, let to = dragToIndex else { return 0 }
            if chore.id == liftedID { return dragOffset }
            if from < to && index > from && index <= to { return -Self.step }
            if from > to && index >= to && index < from { return Self.step }
            return 0
        }()

        let press = LongPressGesture(minimumDuration: 0.38)
            .onChanged { _ in
                if liftedID == nil { pressedID = chore.id }
            }
            .onEnded { _ in
                beginLift(choreID: chore.id, index: index)
            }

        let drag = DragGesture(minimumDistance: 0)
            .onChanged { value in
                guard liftedID == chore.id, let from = dragFromIndex else { return }
                let count = items.count
                let lo = -CGFloat(from) * Self.step - 20
                let hi = CGFloat(count - 1 - from) * Self.step + 20
                let clamped = min(max(value.translation.height, lo), hi)
                let proposed = from + Int((clamped / Self.step).rounded())
                dragOffset = clamped
                dragToIndex = min(max(proposed, 0), count - 1)
            }
            .onEnded { _ in
                finishDrag()
            }

        return rowContent(chore)
            .scaleEffect(isLifted ? 1.035 : (isPressing ? 0.98 : 1))
            .rotationEffect(.degrees(isLifted ? -0.6 : 0))
            .shadow(color: .black.opacity(isLifted ? 0.18 : 0), radius: isLifted ? 22 : 0, y: isLifted ? 12 : 0)
            .overlay(alignment: .trailing) {
                if isLifted {
                    gripDots
                        .padding(.trailing, 44)
                        .transition(.opacity)
                }
            }
            .offset(y: yShift)
            .zIndex(isLifted ? 10 : 0)
            .animation(isLifted ? nil : .spring(response: 0.28, dampingFraction: 0.78), value: yShift)
            .animation(.spring(response: 0.3, dampingFraction: 0.65), value: isLifted)
            .animation(.spring(response: 0.3, dampingFraction: 0.85), value: isPressing)
            .simultaneousGesture(press.sequenced(before: drag))
    }

    private var gripDots: some View {
        VStack(spacing: 4) {
            ForEach(0..<3, id: \.self) { _ in
                HStack(spacing: 4) {
                    Circle().frame(width: 3, height: 3)
                    Circle().frame(width: 3, height: 3)
                }
            }
        }
        .foregroundStyle(Color.hearthTextSecondary)
    }

    private func beginLift(choreID: String, index: Int) {
        guard liftedID == nil else { return }
        UIImpactFeedbackGenerator(style: .light).impactOccurred()
        pressedID = nil
        liftedID = choreID
        dragFromIndex = index
        dragToIndex = index
        dragOffset = 0
        onLiftChange(true)
    }

    private func finishDrag() {
        guard let from = dragFromIndex, let to = dragToIndex, liftedID != nil else {
            resetDragState()
            return
        }
        UIImpactFeedbackGenerator(style: .medium).impactOccurred()
        if from == to {
            withAnimation(.spring(response: 0.32, dampingFraction: 0.58)) {
                resetDragState()
            }
            onLiftChange(false)
            return
        }

        var reordered = items
        let moved = reordered.remove(at: from)
        reordered.insert(moved, at: to)
        let newIDs = reordered.map(\.id)

        withAnimation(.spring(response: 0.32, dampingFraction: 0.58)) {
            resetDragState()
        }
        onLiftChange(false)

        if !hasUsedReorder {
            hasUsedReorder = true
            UserDefaults.standard.set(true, forKey: "hasUsedReorder")
        }

        Task { await onReorder(newIDs) }
    }

    private func resetDragState() {
        liftedID = nil
        pressedID = nil
        dragFromIndex = nil
        dragToIndex = nil
        dragOffset = 0
    }
}
