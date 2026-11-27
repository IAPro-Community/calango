import 'dart:convert';

import 'package:http/http.dart' as http;

import 'generated/models.dart' show ErrorBody;

/// Error returned by the node: the HTTP status plus the stable `error.code`
/// and `error.message` from the API error envelope.
final class CalangoApiException implements Exception {
  const CalangoApiException(this.statusCode, this.code, this.message);

  final int statusCode;

  /// Stable error code, for example `not_found` or `approval_required`.
  /// `internal` when the body is not a valid error envelope.
  final String code;

  final String message;

  @override
  String toString() => 'CalangoApiException($statusCode, $code): $message';
}

/// HTTP transport used by the generated [CalangoClient].
///
/// The token is kept in memory only and sent as a bearer credential on
/// authenticated operations; it is never logged or included in errors.
final class CalangoTransport {
  CalangoTransport({
    required Uri baseUrl,
    String? token,
    http.Client? httpClient,
  }) : _baseUrl = baseUrl,
       _token = token,
       _client = httpClient ?? http.Client(),
       _ownsClient = httpClient == null;

  final Uri _baseUrl;
  final String? _token;
  final http.Client _client;
  final bool _ownsClient;

  /// Sends a request and decodes a JSON response; returns null for an empty body.
  Future<Object?> sendJson(
    String method,
    String path, {
    Map<String, String?> query = const <String, String?>{},
    Map<String, String> headers = const <String, String>{},
    Object? body,
    bool authenticated = true,
  }) async {
    final response = await _send(
      method,
      path,
      query,
      headers,
      body,
      authenticated,
    );
    if (response.statusCode == 204 || response.bodyBytes.isEmpty) return null;
    return jsonDecode(utf8.decode(response.bodyBytes));
  }

  /// Sends a request and returns the response body as text.
  Future<String> sendText(
    String method,
    String path, {
    Map<String, String?> query = const <String, String?>{},
    Map<String, String> headers = const <String, String>{},
    Object? body,
    bool authenticated = true,
  }) async {
    final response = await _send(
      method,
      path,
      query,
      headers,
      body,
      authenticated,
    );
    return utf8.decode(response.bodyBytes);
  }

  /// Closes the HTTP client when this transport created it.
  void close() {
    if (_ownsClient) _client.close();
  }

  Future<http.Response> _send(
    String method,
    String path,
    Map<String, String?> query,
    Map<String, String> headers,
    Object? body,
    bool authenticated,
  ) async {
    final request = http.Request(method, _resolve(path, query));
    final token = _token;
    if (authenticated && token != null) {
      request.headers['Authorization'] = 'Bearer $token';
    }
    request.headers.addAll(headers);
    if (body != null) {
      request.headers['Content-Type'] = 'application/json';
      request.bodyBytes = utf8.encode(jsonEncode(body));
    }
    final response = await http.Response.fromStream(
      await _client.send(request),
    );
    if (response.statusCode < 200 || response.statusCode >= 300) {
      throw _error(response);
    }
    return response;
  }

  Uri _resolve(String path, Map<String, String?> query) {
    final parameters = <String, String>{
      for (final entry in query.entries)
        if (entry.value != null) entry.key: entry.value!,
    };
    final basePath = _baseUrl.path.endsWith('/')
        ? _baseUrl.path.substring(0, _baseUrl.path.length - 1)
        : _baseUrl.path;
    return _baseUrl.replace(
      path: '$basePath$path',
      queryParameters: parameters.isEmpty ? null : parameters,
    );
  }

  CalangoApiException _error(http.Response response) {
    var code = 'internal';
    var message = response.reasonPhrase ?? 'HTTP ${response.statusCode}';
    try {
      final decoded = jsonDecode(utf8.decode(response.bodyBytes));
      if (decoded is Map<String, Object?>) {
        final error = ErrorBody.fromJson(decoded).error;
        code = error.code.value;
        message = error.message;
      }
    } on FormatException {
      // A body that is not a known error envelope keeps the status text.
    } on TypeError {
      // Same as above: the envelope has an unexpected shape.
    }
    return CalangoApiException(response.statusCode, code, message);
  }
}
