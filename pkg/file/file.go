package file

import (
	"webssh-go/app/api/params"
)

type fileHandle struct {
}

var FileHandle = new(fileHandle)

func (f *fileHandle) ListFile(itemInfo params.ItemInfo) {

}
