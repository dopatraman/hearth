import Foundation
import Testing
@testable import HearthApp

@Suite("Smoke")
struct SmokeTests {
    @Test("HealthResponse decodes server JSON")
    func healthResponseDecoding() throws {
        let json = Data(#"{"status":"ok"}"#.utf8)
        let decoded = try JSONDecoder().decode(HealthResponse.self, from: json)
        #expect(decoded == HealthResponse(status: "ok"))
    }
}
