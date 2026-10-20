"""Build TouchSwitch's small, resolution-aware icons from clean vector shapes.
No generative imagery, no shipped fonts, no external image assets required.
"""
from pathlib import Path
from io import BytesIO
import cairosvg
from PIL import Image

base = Path(__file__).resolve().parent
# Clear, geometric finger-pointer icon; no ripples/glow/gradients.
# The background/outline are kept simple so they read at 16 pixels.
hand = '''<g fill="none" stroke="white" stroke-width="8.1" stroke-linecap="round" stroke-linejoin="round">
<path d="M41 59V35c0-4.5 3.1-7.5 7.2-7.5 4.1 0 7.1 3 7.1 7.5v19"/>
<path d="M55.3 48c0-4.1 2.5-6.5 6-6.5s6 2.7 6 6.5v7"/>
<path d="M67.3 52c.1-3.9 2.8-6.1 6.1-6.1s5.8 2.5 5.8 6.1v16c0 10.5-7.4 18-18.1 18H48.7c-5.7 0-8.5-2.1-11.4-5.2L26.2 69c-2.5-2.6-2.2-6 .6-8.4 2.6-2.1 6-1.9 8.5.8L41 67"/>
</g>'''

def render(bg, sz):
    svg=f'''<svg xmlns="http://www.w3.org/2000/svg" width="108" height="108" viewBox="0 0 108 108">
      <rect x="1" y="1" width="106" height="106" rx="25" fill="{bg}"/>
      {hand}
    </svg>'''
    oversample=max(4,min(16,256//sz))
    raw=cairosvg.svg2png(bytestring=svg.encode(),output_width=sz*oversample,output_height=sz*oversample)
    return Image.open(BytesIO(raw)).convert('RGBA').resize((sz,sz),Image.Resampling.LANCZOS)

def save_ico(name,bg):
    sizes=[16,20,24,32,40,48,64,128,256]
    # Pillow ICO writer downsizes source; use per-size source by writing a proper ICO directory manually.
    import struct
    entries=[]
    blobs=[]
    offset=6+16*len(sizes)
    for sz in sizes:
        stream=BytesIO()
        render(bg,sz).save(stream,format='PNG')
        data=stream.getvalue()
        entries.append(struct.pack('<BBBBHHII', sz if sz<256 else 0, sz if sz<256 else 0,0,0,1,32,len(data),offset))
        blobs.append(data);offset+=len(data)
    (base/name).write_bytes(struct.pack('<HHH',0,1,len(sizes))+b''.join(entries)+b''.join(blobs))

save_ico('touch-on.ico','#2563EB')
save_ico('touch-off.ico','#687384')
render('#2563EB',256).save(base/'app-icon.png')
print('Rebuilt on/off ICO assets at 9 native sizes')

# Same white finger graphic used by the tray, main control and 1-second popup.
# Only the tile background is removed for use inside larger controls.
def render_hand_only(stroke, sz):
    s=f'''<svg xmlns="http://www.w3.org/2000/svg" width="108" height="108" viewBox="0 0 108 108">
    {hand.replace('stroke="white"',f'stroke="{stroke}"')}
    </svg>'''
    oversample=max(4,min(16,256//sz))
    raw=cairosvg.svg2png(bytestring=s.encode(),output_width=sz*oversample,output_height=sz*oversample)
    return Image.open(BytesIO(raw)).convert('RGBA').resize((sz,sz),Image.Resampling.LANCZOS)

def save_transparent(name,stroke):
    import struct
    sizes=[16,20,24,32,40,48,64,96,128,256]
    entries=[];blobs=[];offset=6+16*len(sizes)
    for sz in sizes:
        stream=BytesIO()
        render_hand_only(stroke,sz).save(stream,format='PNG')
        data=stream.getvalue()
        entries.append(struct.pack('<BBBBHHII',sz if sz<256 else 0,sz if sz<256 else 0,0,0,1,32,len(data),offset))
        blobs.append(data);offset+=len(data)
    (base/name).write_bytes(struct.pack('<HHH',0,1,len(sizes))+b''.join(entries)+b''.join(blobs))
save_transparent('touch-mark.ico','#FFFFFF')
save_transparent('touch-mark-dark.ico','#191919')
