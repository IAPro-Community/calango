import 'dart:convert';

import 'package:calango_sdk/calango_sdk.dart';
import 'package:test/test.dart';

void main() {
  test('Routine decodes and encodes the wire format', () {
    final wire = <String, Object?>{
      'id': 'rt_1',
      'agent_id': 'ag_1',
      'name': 'Morning summary',
      'prompt': 'Summarize my inbox',
      'trigger_kind': 'cron',
      'schedule': '0 8 * * 1-5',
      'run_on': 'pc',
      'quiet_start': null,
      'notify_on': 'failure',
      'max_retries': 2,
      'paused': false,
      'ended_at': null,
      'next_run_at': '2026-11-26T08:00:00.000Z',
    };

    final routine = Routine.fromJson(
      jsonDecode(jsonEncode(wire)) as Map<String, Object?>,
    );

    expect(routine.triggerKind, RoutineTriggerKind.cron);
    expect(routine.runOn, RunOn.pc);
    expect(routine.quietStart, isNull);
    expect(routine.nextRunAt, DateTime.utc(2026, 11, 26, 8));
    expect(routine.toJson(), <String, Object?>{
      'id': 'rt_1',
      'agent_id': 'ag_1',
      'name': 'Morning summary',
      'prompt': 'Summarize my inbox',
      'trigger_kind': 'cron',
      'schedule': '0 8 * * 1-5',
      'run_on': 'pc',
      'notify_on': 'failure',
      'max_retries': 2,
      'paused': false,
      'next_run_at': '2026-11-26T08:00:00.000Z',
    });
  });

  test('allOf schemas carry every part', () {
    final detail = RunDetail.fromJson(<String, Object?>{
      'id': 'run_1',
      'agent_id': 'ag_1',
      'conversation_id': 'cv_1',
      'source_message_id': 'msg_1',
      'trigger': 'message',
      'status': 'waiting_approval',
      'tokens_in': 10,
      'tokens_out': 20,
      'steps': <Object?>[
        <String, Object?>{
          'id': 'st_1',
          'seq': 1,
          'kind': 'tool_call',
          'payload': '{}',
          'created_at': '2026-11-25T10:00:00Z',
        },
      ],
    });

    expect(detail.status, RunStatus.waitingApproval);
    expect(detail.steps.single.seq, 1);
    expect(
      (detail.toJson()['steps'] as List<Object?>).single,
      containsPair('created_at', '2026-11-25T10:00:00.000Z'),
    );
  });

  test('required nullable fields are sent as null', () {
    const verification = AuditVerification(
      ok: true,
      entriesChecked: 3,
      firstBreak: null,
    );

    expect(verification.toJson(), <String, Object?>{
      'ok': true,
      'entries_checked': 3,
      'first_break': null,
    });
  });

  test('unknown enum values are rejected', () {
    expect(() => RunStatus.fromJson('paused'), throwsFormatException);
  });
}
