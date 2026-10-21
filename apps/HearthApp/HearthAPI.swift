import Foundation

// Minimal URLSession client stub. The real client (auth, retries, decoding
// strategy, error mapping) lands incrementally starting in Slice 2.

struct HearthAPI {
    let baseURL: URL
    let session: URLSession
    var bearerToken: String?

    init(baseURL: URL, session: URLSession = .shared, bearerToken: String? = nil) {
        self.baseURL = baseURL
        self.session = session
        self.bearerToken = bearerToken
    }

    func request(_ path: String, method: String = "GET") -> URLRequest {
        var req = URLRequest(url: baseURL.appendingPathComponent(path))
        req.httpMethod = method
        if let token = bearerToken {
            req.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
        }
        return req
    }
}
