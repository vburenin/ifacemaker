package mypackage

type LocalType struct {
    Value int
}

type MyStruct struct{}

func (m *MyStruct) UseLocal(x LocalType) LocalType {
    return x
}

func (m *MyStruct) UseBuiltin(x int) string {
    return "test"
}

func (m *MyStruct) UsePointer(x *LocalType) *LocalType {
    return x
}

func (m *MyStruct) UseSlice(x []LocalType) []LocalType {
    return x
}

func (m *MyStruct) UseMap(x map[string]LocalType) map[LocalType]string {
    return nil
}
