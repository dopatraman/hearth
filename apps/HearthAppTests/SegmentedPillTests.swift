import Foundation
import Testing
@testable import HearthApp

@Suite("segmentIndex")
struct SegmentedPillTests {
    enum TestOption: Hashable { case alpha, beta, gamma }

    @Test("first option returns 0")
    func segmentIndex_firstOption_returnsZero() {
        let options: [TestOption] = [.alpha, .beta]
        #expect(segmentIndex(selected: TestOption.alpha, options: options) == 0)
    }

    @Test("second option returns 1")
    func segmentIndex_secondOption_returnsOne() {
        let options: [TestOption] = [.alpha, .beta]
        #expect(segmentIndex(selected: TestOption.beta, options: options) == 1)
    }

    @Test("unknown option returns 0 (sensible fallback)")
    func segmentIndex_unknownOption_returnsZero() {
        let options: [TestOption] = [.alpha, .beta]
        #expect(segmentIndex(selected: TestOption.gamma, options: options) == 0)
    }

    @Test("empty options returns 0")
    func segmentIndex_emptyOptions_returnsZero() {
        let options: [TestOption] = []
        #expect(segmentIndex(selected: TestOption.alpha, options: options) == 0)
    }
}
