//
//  WebSocketService.swift
//  TorrentSeeker
//
//  WebSocket service for real-time updates
//

import Foundation

class WebSocketService: NSObject, ObservableObject {
    @Published var isConnected = false

    private var webSocketTask: URLSessionWebSocketTask?
    private var token: String?

    var onWorkerStatus: ((String, String?) -> Void)?
    var onNewMatch: ((Match) -> Void)?
    var onNewLog: ((Log) -> Void)?

    func connect(token: String, host: String = "127.0.0.1:8080") {
        self.token = token

        // Determine protocol based on whether we're using SSL
        let urlString = "ws://\(host)/api/ws?token=\(token)"

        guard let url = Foundation.URL(string: urlString) else {
            print("Invalid WebSocket URL")
            return
        }

        let session = URLSession(configuration: .default, delegate: self, delegateQueue: OperationQueue())
        webSocketTask = session.webSocketTask(with: url)
        webSocketTask?.resume()

        receiveMessage()
    }

    func disconnect() {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        isConnected = false
    }

    private func receiveMessage() {
        webSocketTask?.receive { [weak self] result in
            switch result {
            case .success(let message):
                switch message {
                case .string(let text):
                    self?.handleMessage(text)
                case .data(let data):
                    if let text = String(data: data, encoding: .utf8) {
                        self?.handleMessage(text)
                    }
                @unknown default:
                    break
                }

                // Continue receiving messages
                self?.receiveMessage()

            case .failure(let error):
                print("WebSocket error: \(error)")
                self?.isConnected = false
            }
        }
    }

    private func handleMessage(_ text: String) {
        guard let data = text.data(using: .utf8) else { return }

        do {
            let message = try JSONDecoder().decode(WebSocketMessage.self, from: data)

            DispatchQueue.main.async {
                switch message.type {
                case "worker_status":
                    print("Worker status: \(message.status ?? ""), \(message.message ?? "")")
                    self.onWorkerStatus?(message.status ?? "", message.message)

                case "new_match":
                    if let match = message.match {
                        print("New match: \(match)")
                        self.onNewMatch?(match)
                    }

                case "new_log":
                    if let log = message.log {
                        print("New log: \(log)")
                        self.onNewLog?(log)
                    }

                default:
                    print("Unknown message type: \(message.type)")
                }
            }
        } catch {
            print("Failed to decode WebSocket message: \(error)")
        }
    }
}

extension WebSocketService: URLSessionWebSocketDelegate {
    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didOpenWithProtocol protocol: String?) {
        print("WebSocket connected")
        DispatchQueue.main.async {
            self.isConnected = true
        }
    }

    func urlSession(_ session: URLSession, webSocketTask: URLSessionWebSocketTask, didCloseWith closeCode: URLSessionWebSocketTask.CloseCode, reason: Data?) {
        print("WebSocket disconnected")
        DispatchQueue.main.async {
            self.isConnected = false
        }
    }
}
