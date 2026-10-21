import Foundation

// Placeholder types that mirror the Go server's JSON contract.
// Fleshed out slice by slice as the API grows (first real use: Slice 3).

struct HealthResponse: Codable, Equatable {
    let status: String
}
