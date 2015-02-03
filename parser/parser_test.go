package parser

import (
	"bufio"
	"os"
	"strings"
	"testing"
)

func TestParseOneFrame(t *testing.T) {
	frame := `    ffffffff8149767f __schedule ([kernel.kallsyms])
    ffffffff810901eb sys_sched_yield ([kernel.kallsyms])
    ffffffff814a41a9 system_call_fastpath ([kernel.kallsyms])
        7f5880a7db97 __sched_yield (/lib64/libc-2.17.so)
             134789c __wt_log_write (/data/3.0/rc7d/bin/mongod)
             139345e __wt_txn_commit (/data/3.0/rc7d/bin/mongod)
             1389929 __session_commit_transaction (/data/3.0/rc7d/bin/mongod)
              d6d313 mongo::WiredTigerRecoveryUnit::_txnClose(bool) (/data/3.0/rc7d/bin/mongod)
              d6d69a mongo::WiredTigerRecoveryUnit::_commit() (/data/3.0/rc7d/bin/mongod)
              9133e4 mongo::WriteUnitOfWork::commit() (/data/3.0/rc7d/bin/mongod)
              9addc1 mongo::WriteBatchExecutor::execOneInsert(mongo::WriteBatchExecutor::ExecInsertsState*, mongo::WriteErrorDetail**) (/data/3.0/rc7d/bin/mongod)
              9ae8a6 mongo::WriteBatchExecutor::execInserts(mongo::BatchedCommandRequest const&, std::vector<mongo::WriteErrorDetail*, std::allocator<mongo::WriteErro>
              9ae987 mongo::WriteBatchExecutor::bulkExecute(mongo::BatchedCommandRequest const&, std::vector<mongo::BatchedUpsertDetail*, std::allocator<mongo::Batche>
              9af091 mongo::WriteBatchExecutor::executeBatch(mongo::BatchedCommandRequest const&, mongo::BatchedCommandResponse*) (/data/3.0/rc7d/bin/mongod)
              9b1c5c mongo::WriteCmd::run(mongo::OperationContext*, std::string const&, mongo::BSONObj&, int, std::string&, mongo::BSONObjBuilder&, bool) (/data/3.0/r>
              9d1531 mongo::_execCommand(mongo::OperationContext*, mongo::Command*, std::string const&, mongo::BSONObj&, int, std::string&, mongo::BSONObjBuilder&, bo>
              9d2508 mongo::Command::execCommand(mongo::OperationContext*, mongo::Command*, int, char const*, mongo::BSONObj&, mongo::BSONObjBuilder&, bool) (/data/3.>
              9d30eb mongo::_runCommands(mongo::OperationContext*, char const*, mongo::BSONObj&, mongo::_BufBuilder<mongo::TrivialAllocator>&, mongo::BSONObjBuilder&,>
              b9aabe mongo::runQuery(mongo::OperationContext*, mongo::Message&, mongo::QueryMessage&, mongo::NamespaceString const&, mongo::CurOp&, mongo::Message&, b>
              aafb19 mongo::assembleResponse(mongo::OperationContext*, mongo::Message&, mongo::DbResponse&, mongo::HostAndPort const&, bool) (/data/3.0/rc7d/bin/mongo>
              806f08 mongo::MyMessageHandler::process(mongo::Message&, mongo::AbstractMessagingPort*, mongo::LastError*) (/data/3.0/rc7d/bin/mongod)
              f05a59 mongo::PortMessageServer::handleIncomingMsg(void*) (/data/3.0/rc7d/bin/mongod)
        7f5881982f18 start_thread (/lib64/libpthread-2.17.so)

`
	pf, err := parseOneFrame(bufio.NewScanner(strings.NewReader(frame)), "mongod 28451 10835064.656792: cpu-clock:")

	if err != nil {
		t.Error("Receive error from parseOneFrame: ", err)
	}

	if pf.Process != "mongod" {
		t.Errorf("perfFrame has wrong name, expecting [%10s] got [%10s]", "mongod", pf.Process)
	}

	if pf.Pid != 28451 {
		t.Errorf("perfFrame has wrong pid, expecting [%10d] got [%10d]", 28451, pf.Pid)
	}

	if pf.TS != 10835064.656792 {
		t.Errorf("perfFrame has wrong pid, expecting [%10.2f] got [%10.2f]", 10835064.656792, pf.TS)
	}

	if pf.Functions[0].Function != "__schedule" {
		t.Errorf("perfFrame Functions[0] is not __schedule, got [%10s]", pf.Functions[0].Function)
	}

	if pf.Functions[10].Function != `mongo::WriteBatchExecutor::execOneInsert(mongo::WriteBatchExecutor::ExecInsertsState*, mongo::WriteErrorDetail**)` {
		t.Errorf("perfFrame Functions[0] is wrong, got [%10s]", pf.Functions[10].Function)
	}

	if pf.Functions[22].Function != `start_thread` {
		t.Errorf("perfFrame Functions[0] is wrong, got [%10s]", pf.Functions[22].Function)
	}

	if pf.Functions[21].ExecutionSpace != `(/data/3.0/rc7d/bin/mongod)` {
		t.Errorf("perfFrame Functions[0] is wrong, got [%10s]", pf.Functions[21].ExecutionSpace)
	}

	if len(pf.Functions) != 23 {
		t.Errorf("length of stack is not %d, got %d\n", 23, len(pf.Functions))
	}
}

func TestParsePerfScript(t *testing.T) {
	f, err := os.Open("perf.script.test")

	if err != nil {
		t.Error("Failed to open perf.script")
	}

	frames, e := ParsePerfScript(f)

	if e != nil {
		t.Error("ParsePerfScript return error: ", e)
	}

	if len(frames) != 92 {
		t.Errorf("Frames length error, expecting 5 got %d", len(frames))
	}

	if len(frames[0].Functions) != 23 {
		t.Errorf("First frame shall have 23 function, got %d", len(frames[0].Functions))
		t.FailNow()
	}

	if frames[1].TS != 10835064.656850 {
		t.Errorf("TS for frame 1 expectig %10.2f, got %10.2f", 10835064.656850, frames[1].TS)
	}
}

func TestParsePerfScriptTimeline(t *testing.T) {
	f, err := os.Open("perf.script.test2")

	if err != nil {
		t.Error("Failed to open perf.script")
		t.FailNow()
	}

	timeline, e := ParsePerfScriptTimeline(f)

	if e != nil {
		t.Error("ParsePerfScriptTimeline return error: ", err)
		t.FailNow()
	}

	if len(timeline) != 5 {
		t.Errorf("Timeline shall be for 5 intervals, got %d", len(timeline))
	}

	if timeline[0].NumSample != 1354 {
		t.Errorf("Timeslot 1 shall have 1354 samples, got %d", timeline[0].NumSample)
	}

	PrintPerfTimeline(timeline)
	//	t.Error("I want print")
}
