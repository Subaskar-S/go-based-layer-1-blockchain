package vm

import (
	"context"
	"encoding/binary"
	"fmt"

	"github.com/blockchain/layer1/internal/crypto"
	"github.com/blockchain/layer1/internal/state"
	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/api"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
)

const (
	// MaxGasLimit is the maximum gas allowed for a contract execution
	MaxGasLimit = 10_000_000
	// GasPerOp is the base gas cost per WASM operation
	GasPerOp = 1
	// GasPerMemoryByte is the gas cost per byte of memory
	GasPerMemoryByte = 1
)

// VM represents the WASM virtual machine
type VM struct {
	runtime wazero.Runtime
	ctx     context.Context
	stateDB *state.StateDB
	
	// Execution context
	caller       crypto.Address
	contractAddr crypto.Address
	gasLimit     uint64
	gasUsed      uint64
	logs         []Log
}

// Log represents an event log from contract execution
type Log struct {
	Address crypto.Address
	Topics  []crypto.Hash
	Data    []byte
}

// ExecutionResult represents the result of contract execution
type ExecutionResult struct {
	ReturnData []byte
	GasUsed    uint64
	Logs       []Log
	Error      error
}

// NewVM creates a new WASM VM
func NewVM(ctx context.Context, stateDB *state.StateDB) (*VM, error) {
	// Create wazero runtime with default configuration
	runtime := wazero.NewRuntime(ctx)
	
	// Instantiate WASI for basic I/O (optional, for debugging)
	if _, err := wasi_snapshot_preview1.Instantiate(ctx, runtime); err != nil {
		return nil, fmt.Errorf("failed to instantiate WASI: %w", err)
	}
	
	return &VM{
		runtime: runtime,
		ctx:     ctx,
		stateDB: stateDB,
		logs:    make([]Log, 0),
	}, nil
}

// Close closes the VM
func (vm *VM) Close() error {
	return vm.runtime.Close(vm.ctx)
}

// DeployContract deploys a new contract
func (vm *VM) DeployContract(caller crypto.Address, code []byte, gasLimit uint64) (*ExecutionResult, crypto.Address, error) {
	// Generate contract address (simplified: hash of caller + nonce)
	nonce, err := vm.stateDB.GetNonce(caller)
	if err != nil {
		return nil, crypto.ZeroAddress(), err
	}
	
	addrData := append(caller.Bytes(), byte(nonce))
	contractAddr := crypto.Address{}
	hash := crypto.HashData(addrData)
	copy(contractAddr[:], hash[:20])
	
	// Store contract code
	codeHash := crypto.HashData(code)
	if err := vm.stateDB.SetCode(codeHash, code); err != nil {
		return nil, crypto.ZeroAddress(), err
	}
	
	// Create contract account
	account, err := vm.stateDB.GetAccount(contractAddr)
	if err != nil {
		return nil, crypto.ZeroAddress(), err
	}
	account.IsContract = true
	account.CodeHash = codeHash
	
	if err := vm.stateDB.SetAccount(account); err != nil {
		return nil, crypto.ZeroAddress(), err
	}
	
	// Execute constructor if present
	result := vm.Execute(caller, contractAddr, []byte{}, gasLimit, 0)
	
	return result, contractAddr, result.Error
}

// Execute executes a contract call
func (vm *VM) Execute(caller, contractAddr crypto.Address, input []byte, gasLimit, value uint64) *ExecutionResult {
	vm.caller = caller
	vm.contractAddr = contractAddr
	vm.gasLimit = gasLimit
	vm.gasUsed = 0
	vm.logs = make([]Log, 0)
	
	// Get contract account
	account, err := vm.stateDB.GetAccount(contractAddr)
	if err != nil {
		return &ExecutionResult{Error: err}
	}
	
	if !account.IsContract {
		return &ExecutionResult{Error: fmt.Errorf("address is not a contract")}
	}
	
	// Get contract code
	code, err := vm.stateDB.GetCode(account.CodeHash)
	if err != nil {
		return &ExecutionResult{Error: err}
	}
	
	// Transfer value if any
	if value > 0 {
		if err := vm.stateDB.Transfer(caller, contractAddr, value); err != nil {
			return &ExecutionResult{Error: err}
		}
	}
	
	// Compile and execute WASM module
	returnData, err := vm.executeWASM(code, input)
	
	return &ExecutionResult{
		ReturnData: returnData,
		GasUsed:    vm.gasUsed,
		Logs:       vm.logs,
		Error:      err,
	}
}

// executeWASM executes WASM code
func (vm *VM) executeWASM(code, input []byte) ([]byte, error) {
	// Build host module with environment functions
	hostModule := vm.runtime.NewHostModuleBuilder("env")
	
	// Register host functions
	hostModule = hostModule.
		NewFunctionBuilder().
		WithFunc(vm.hostGetBalance).
		Export("get_balance").
		NewFunctionBuilder().
		WithFunc(vm.hostTransfer).
		Export("transfer").
		NewFunctionBuilder().
		WithFunc(vm.hostStorageGet).
		Export("storage_get").
		NewFunctionBuilder().
		WithFunc(vm.hostStorageSet).
		Export("storage_set").
		NewFunctionBuilder().
		WithFunc(vm.hostEmitLog).
		Export("emit_log").
		NewFunctionBuilder().
		WithFunc(vm.hostGetCaller).
		Export("get_caller").
		NewFunctionBuilder().
		WithFunc(vm.hostGetCallValue).
		Export("get_call_value")
	
	// Instantiate host module
	if _, err := hostModule.Instantiate(vm.ctx); err != nil {
		return nil, fmt.Errorf("failed to instantiate host module: %w", err)
	}
	
	// Compile WASM module
	compiledModule, err := vm.runtime.CompileModule(vm.ctx, code)
	if err != nil {
		return nil, fmt.Errorf("failed to compile WASM: %w", err)
	}
	defer compiledModule.Close(vm.ctx)
	
	// Instantiate module
	module, err := vm.runtime.InstantiateModule(vm.ctx, compiledModule, wazero.NewModuleConfig())
	if err != nil {
		return nil, fmt.Errorf("failed to instantiate module: %w", err)
	}
	defer module.Close(vm.ctx)
	
	// Get the main function (entry point)
	mainFn := module.ExportedFunction("main")
	if mainFn == nil {
		return nil, fmt.Errorf("main function not found")
	}
	
	// Allocate memory for input (if module has memory)
	memory := module.Memory()
	if memory == nil {
		return nil, fmt.Errorf("module has no memory")
	}
	
	// Write input to memory
	inputPtr := uint32(1024) // Start at offset 1024
	if !memory.Write(inputPtr, input) {
		return nil, fmt.Errorf("failed to write input to memory")
	}
	
	// Execute main function with input pointer and length
	results, err := mainFn.Call(vm.ctx, uint64(inputPtr), uint64(len(input)))
	if err != nil {
		return nil, fmt.Errorf("execution failed: %w", err)
	}
	
	// Read return data from memory
	if len(results) < 2 {
		return []byte{}, nil
	}
	
	returnPtr := uint32(results[0])
	returnLen := uint32(results[1])
	
	returnData, ok := memory.Read(returnPtr, returnLen)
	if !ok {
		return nil, fmt.Errorf("failed to read return data")
	}
	
	return returnData, nil
}

// Host functions

func (vm *VM) hostGetBalance(ctx context.Context, m api.Module, addrPtr, addrLen uint32) uint64 {
	vm.consumeGas(100)
	
	memory := m.Memory()
	addrBytes, ok := memory.Read(addrPtr, addrLen)
	if !ok || len(addrBytes) != 20 {
		return 0
	}
	
	addr, _ := crypto.AddressFromBytes(addrBytes)
	balance, _ := vm.stateDB.GetBalance(addr)
	
	return balance
}

func (vm *VM) hostTransfer(ctx context.Context, m api.Module, toPtr, toLen uint32, amount uint64) uint32 {
	vm.consumeGas(1000)
	
	memory := m.Memory()
	toBytes, ok := memory.Read(toPtr, toLen)
	if !ok || len(toBytes) != 20 {
		return 1 // Error
	}
	
	to, _ := crypto.AddressFromBytes(toBytes)
	
	if err := vm.stateDB.Transfer(vm.contractAddr, to, amount); err != nil {
		return 1 // Error
	}
	
	return 0 // Success
}

func (vm *VM) hostStorageGet(ctx context.Context, m api.Module, keyPtr, keyLen, valuePtr uint32) uint32 {
	vm.consumeGas(200)
	
	memory := m.Memory()
	key, ok := memory.Read(keyPtr, keyLen)
	if !ok {
		return 0
	}
	
	value, _ := vm.stateDB.GetStorage(vm.contractAddr, key)
	
	if len(value) > 0 {
		memory.Write(valuePtr, value)
		return uint32(len(value))
	}
	
	return 0
}

func (vm *VM) hostStorageSet(ctx context.Context, m api.Module, keyPtr, keyLen, valuePtr, valueLen uint32) uint32 {
	vm.consumeGas(500)
	
	memory := m.Memory()
	key, ok := memory.Read(keyPtr, keyLen)
	if !ok {
		return 1
	}
	
	value, ok := memory.Read(valuePtr, valueLen)
	if !ok {
		return 1
	}
	
	if err := vm.stateDB.SetStorage(vm.contractAddr, key, value); err != nil {
		return 1
	}
	
	return 0
}

func (vm *VM) hostEmitLog(ctx context.Context, m api.Module, dataPtr, dataLen uint32) {
	vm.consumeGas(300)
	
	memory := m.Memory()
	data, ok := memory.Read(dataPtr, dataLen)
	if !ok {
		return
	}
	
	log := Log{
		Address: vm.contractAddr,
		Topics:  []crypto.Hash{},
		Data:    data,
	}
	
	vm.logs = append(vm.logs, log)
}

func (vm *VM) hostGetCaller(ctx context.Context, m api.Module, outPtr uint32) {
	vm.consumeGas(50)
	
	memory := m.Memory()
	memory.Write(outPtr, vm.caller.Bytes())
}

func (vm *VM) hostGetCallValue(ctx context.Context, m api.Module) uint64 {
	vm.consumeGas(50)
	return 0 // Would track call value in execution context
}

// consumeGas consumes gas and checks limit
func (vm *VM) consumeGas(amount uint64) {
	vm.gasUsed += amount
	if vm.gasUsed > vm.gasLimit {
		panic("out of gas")
	}
}

// Helper to write uint64 to memory
func writeUint64(memory api.Memory, ptr uint32, value uint64) {
	buf := make([]byte, 8)
	binary.LittleEndian.PutUint64(buf, value)
	memory.Write(ptr, buf)
}

