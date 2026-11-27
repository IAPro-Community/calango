import 'dart:convert';

import 'package:calango_sdk/calango_sdk.dart';
import 'package:http/http.dart' as http;
import 'package:http/testing.dart';
import 'package:test/test.dart';

CalangoClient clientFor(
  Future<http.Response> Function(http.Request request) handler, {
  String? token = 'secret-token',
}) {
  return CalangoClient(
    CalangoTransport(
      baseUrl: Uri.parse('http://127.0.0.1:7317'),
      token: token,
      httpClient: MockClient(handler),
    ),
  );
}

http.Response json(Object? body, int status) => http.Response(
  jsonEncode(body),
  status,
  headers: <String, String>{'content-type': 'application/json'},
);

void main() {
  test('sends the bearer token, query and decodes a list', () async {
    late http.Request seen;
    final client = clientFor((request) async {
      seen = request;
      return json(<Object?>[
        <String, Object?>{
          'id': 'ap_1',
          'agent_id': 'ag_1',
          'action': 'email.send',
          'target': 'client@example.com',
          'status': 'pending',
        },
      ], 200);
    });

    final approvals = await client.listApprovals(
      status: ApprovalStatus.pending,
    );

    expect(seen.method, 'GET');
    expect(
      seen.url.toString(),
      'http://127.0.0.1:7317/v1/approvals?status=pending',
    );
    expect(seen.headers['Authorization'], 'Bearer secret-token');
    expect(approvals.single.status, ApprovalStatus.pending);
  });

  test('sends headers and an inline body', () async {
    late http.Request seen;
    final client = clientFor((request) async {
      seen = request;
      return json(<String, Object?>{
        'message': <String, Object?>{
          'id': 'msg_1',
          'conversation_id': 'cv 1',
          'author_kind': 'person',
          'author_id': 'me',
          'body': 'hi',
          'created_at': '2026-11-25T10:00:00Z',
        },
        'run': <String, Object?>{
          'id': 'run_1',
          'agent_id': 'ag_1',
          'conversation_id': 'cv 1',
          'source_message_id': 'msg_1',
          'trigger': 'message',
          'status': 'queued',
          'tokens_in': 0,
          'tokens_out': 0,
        },
      }, 201);
    });

    final created = await client.createMessage(
      'cv 1',
      body: 'hi',
      idempotencyKey: 'k-1',
    );

    expect(seen.url.path, '/v1/conversations/cv%201/messages');
    expect(seen.headers['Idempotency-Key'], 'k-1');
    expect(seen.headers['Content-Type'], startsWith('application/json'));
    expect(jsonDecode(seen.body), <String, Object?>{'body': 'hi'});
    expect(created.run.status, RunStatus.queued);
  });

  test('public operations do not send the token', () async {
    late http.Request seen;
    final client = clientFor((request) async {
      seen = request;
      return json(<String, Object?>{
        'device_id': 'dv_1',
        'token': 't',
        'scope': 'approve_only',
      }, 200);
    });

    final paired = await client.pairDevice(
      code: 'ABCD1234',
      deviceName: 'phone',
      platform: DevicePlatform.android,
    );

    expect(seen.headers.containsKey('Authorization'), isFalse);
    expect(paired.scope, DeviceScope.approveOnly);
  });

  test('maps the error envelope to CalangoApiException', () async {
    final client = clientFor(
      (request) async => json(<String, Object?>{
        'error': <String, Object?>{
          'code': 'not_found',
          'message': 'no such run',
        },
      }, 404),
    );

    await expectLater(
      client.getRun('run_x'),
      throwsA(
        isA<CalangoApiException>()
            .having((e) => e.statusCode, 'statusCode', 404)
            .having((e) => e.code, 'code', 'not_found')
            .having((e) => e.message, 'message', 'no such run')
            .having(
              (e) => e.toString(),
              'toString',
              isNot(contains('secret-token')),
            ),
      ),
    );
  });

  test('empty responses and text responses', () async {
    final client = clientFor((request) async {
      if (request.url.path == '/v1/audit/export') {
        return http.Response(
          '{"seq":1}\n',
          200,
          headers: <String, String>{'content-type': 'application/x-ndjson'},
        );
      }
      return http.Response('', 204);
    });

    await client.deleteAgent('ag_1');
    expect(
      await client.exportAudit(format: ExportAuditFormat.jsonl),
      '{"seq":1}\n',
    );
  });
}
