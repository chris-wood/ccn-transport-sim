package core

import "strings"
import "fmt"

type FibTableEntry struct {
    Prefix string `json:"prefix"`
    Interfaces []int `json:"interface"`
}

type FibTable struct {
    Table map[string]FibTableEntry `json:"table"`
}

func (f FibTable) AddPrefix(prefix string, face int) {
    if val, ok := f.Table[prefix]; ok {
        // TODO: face might already be in the interface list
        entry := FibTableEntry{prefix, append(val.Interfaces, face)};
        f.Table[prefix] = entry;
    } else {
        entry := FibTableEntry{prefix, []int{face}};
        f.Table[prefix] = entry;
    }
}

type NotInFibError struct {
    desc string
}

func (nif NotInFibError) Error() (string) {
    return nif.desc;
}

func (f FibTable) GetInterfacesForPrefix(prefix string) ([]int, error){
    var faces []int;

    components := strings.Split(prefix, "/");
    foundEntry := false;

    // Check via LPM
    for i := 0; i < len(components); i++ {
        fullPrefix := ""
        for j := 0; j < i; j++ {
            fullPrefix = fullPrefix + components[j];
            fullPrefix = fullPrefix + "/";
        }
        fullPrefix = fullPrefix + components[i];

        if val, ok := f.Table[fullPrefix]; ok {
            faces = val.Interfaces;
            foundEntry = true;
        }
    }

    if !foundEntry {
        return faces, NotInFibError{fmt.Sprintf("%s not in FIB", prefix)};
    } else {
        return faces, nil;
    }
}
