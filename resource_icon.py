"""Create a Go .syso COFF icon resource from the same tray ICO.
The resource embeds an actual Explorer/EXE icon for each target architecture;
no windres or network packages required. MIT licensed with TouchSwitch.
Usage: python resource_icon.py amd64|arm64
"""
import struct,sys
from pathlib import Path
base=Path(__file__).parent
arch=sys.argv[1] if len(sys.argv)>1 else 'amd64'
assert arch in ('amd64','arm64')
machine,reloc_type={'amd64':(0x8664,0x0003),'arm64':(0xaa64,0x0002)}[arch]
ico=(base/'touch-on.ico').read_bytes()
reserved,kind,count=struct.unpack_from('<HHH',ico,0)
assert reserved==0 and kind==1 and 0<count<100
icons=[]; group=bytearray(struct.pack('<HHH',0,1,count))
for i in range(count):
    raw=ico[6+16*i:6+16*(i+1)]
    sz,ofs=struct.unpack_from('<II',raw,8)
    icons.append(ico[ofs:ofs+sz])
    group+=raw[:12]+struct.pack('<H',i+1)
# Build section with a resource directory tree, numeric Type>ID>Lang (0x409).
blob=bytearray()
def append(size,align=4):
    while len(blob)%align:blob.append(0)
    pos=len(blob);blob.extend(b'\0'*size);return pos
def write(off,fmt,*v):struct.pack_into(fmt,blob,off,*v)
def dirn(ids):
    pos=append(16+8*len(ids));write(pos,'<IIHHHH',0,0,0,0,0,len(ids))
    return pos

top=dirn([3,14]);write(top+16,'<I',3);write(top+24,'<I',14)
type_icons=dirn(list(range(1,count+1)))
type_group=dirn([1])
write(top+20,'<I',type_icons|0x80000000);write(top+28,'<I',type_group|0x80000000)
leafs=[]
for i in range(count):
    name=dirn([1033]);write(type_icons+16+8*i,'<II',i+1,name|0x80000000)
    leafs.append((name,icons[i]))
group_name=dirn([1033]);write(type_group+16,'<II',1,group_name|0x80000000)
leafs.append((group_name,bytes(group)))
relocs=[]
for name,data in leafs:
    dataEntry=append(16)
    write(name+16,'<II',1033,dataEntry)
    position=append(len(data))
    blob[position:position+len(data)]=data
    write(dataEntry,'<IIII',position,len(data),0,0)
    relocs.append((dataEntry,0,reloc_type))
# A COFF object: 20-byte file hdr + 40-byte section hdr + raw section + relocations + symbol + string table.
raw_offset=60
reloc_offset=raw_offset+len(blob)
sym_offset=reloc_offset+10*len(relocs)
header=struct.pack('<HHIIIHH',machine,1,0,sym_offset,1,0,0)
section=struct.pack('<8sIIIIIIHHI',b'.rsrc\0\0\0',0,0,len(blob),raw_offset,reloc_offset,0,len(relocs),0,0x40000040)
relocation=b''.join(struct.pack('<IIH',off,sym,kind) for off,sym,kind in relocs)
symbol=struct.pack('<8sIhHBB',b'.rsrc\0\0\0',0,1,0,3,0)
stringTable=struct.pack('<I',4)
(base/'rsrc.syso').write_bytes(header+section+blob+relocation+symbol+stringTable)
print(f'{arch}: wrote icon resources {count} sizes, {len(blob)} bytes')
