#!/usr/bin/env python3
"""Prints -actions tokens that build a small test town on seed 23 (centre 64,64).

Usage: ./sprawl -seed 23 -actions "$(python3 tools/town.py r),cancel" -ticks 4000 -screenshot out.png
Extra arguments are appended before the final cursor move.
"""
import sys
cur=[64,64]; out=[]
def goto(x,y):
    dx=x-cur[0]; dy=y-cur[1]
    while abs(dx)>=8: out.append('L' if dx>0 else 'H'); dx-=8 if dx>0 else -8
    while dx: out.append('l' if dx>0 else 'h'); dx-=1 if dx>0 else -1
    while abs(dy)>=8: out.append('J' if dy>0 else 'K'); dy-=8 if dy>0 else -8
    while dy: out.append('j' if dy>0 else 'k'); dy-=1 if dy>0 else -1
    cur[0],cur[1]=x,y
def span(tool,a,b):
    out.extend(tool); goto(*a); out.append('v'); goto(*b); out.append('apply')
span(['r'],(54,62),(78,62)); span(['r'],(66,56),(66,76)); span(['r'],(54,70),(78,70))
span(['z','r'],(55,63),(65,64)); span(['z','r'],(55,68),(65,69))
span(['z','c'],(67,63),(77,64)); span(['z','i'],(67,71),(77,72))
span(['p'],(54,58),(70,58)); span(['p'],(70,58),(70,62))
NAMES={'1':'power plant','3':'water pump','5':'police station','6':'fire station','7':'school'}
def place(menu_key,at):
    out.append('@tool:'+NAMES[menu_key]); goto(*at); out.append('apply')
place('1',(51,57)); place('3',(51,78))
out.extend(['p']); goto(54,69); out.append('v'); goto(53,78); out.append('o'); out.append('apply')
span(['p'],(66,63),(66,71)); span(['w'],(53,78),(60,78)); span(['w'],(60,78),(60,73))
span(['w'],(56,65),(76,65)); span(['w'],(60,60),(60,73))
place('5',(56,59)); place('6',(62,59)); place('7',(57,66))
extra=sys.argv[1:] 
out.extend(extra)
goto(66,66)
print(','.join(out))
