from pathlib import Path
from reportlab.lib.pagesizes import A4
from reportlab.platypus import SimpleDocTemplate, Paragraph, Spacer
from reportlab.lib.styles import getSampleStyleSheet, ParagraphStyle
from reportlab.lib.enums import TA_CENTER
from reportlab.lib import colors
from reportlab.lib.units import mm
import re

src=Path("docs/Go_Data_Profiler_Master_Copy.md")
out=Path("docs/Go_Data_Profiler_Master_Copy.pdf")
text=src.read_text(encoding="utf-8")
s=getSampleStyleSheet()
s.add(ParagraphStyle(name="TC",parent=s["Title"],alignment=TA_CENTER,fontSize=23,leading=28,spaceAfter=12))
s.add(ParagraphStyle(name="SC",parent=s["Normal"],alignment=TA_CENTER,fontSize=10.5,leading=15,spaceAfter=16))
s.add(ParagraphStyle(name="H1M",parent=s["Heading1"],fontSize=16,leading=20,spaceBefore=11,spaceAfter=6))
s.add(ParagraphStyle(name="BM",parent=s["BodyText"],fontSize=9,leading=12.7,spaceAfter=5))
s.add(ParagraphStyle(name="BLM",parent=s["BodyText"],fontSize=8.8,leading=12.5,leftIndent=13,firstLineIndent=-6,spaceAfter=3))
s.add(ParagraphStyle(name="CM",parent=s["Code"],fontName="Courier",fontSize=6.8,leading=8.6,leftIndent=6,rightIndent=6,spaceBefore=3,spaceAfter=5,backColor=colors.HexColor("#F2F2F2")))
def esc(x): return x.replace("&","&amp;").replace("<","&lt;").replace(">","&gt;")
def footer(canvas,doc):
    canvas.saveState(); canvas.setFont("Helvetica",7); canvas.setFillColor(colors.HexColor("#666666"))
    canvas.drawString(17*mm,9*mm,"Go Data Profiler — Master Technical Copy")
    canvas.drawRightString(193*mm,9*mm,f"Page {doc.page}"); canvas.restoreState()
story=[]
lines=text.splitlines()
i=0
while i<len(lines):
    line=lines[i]
    if line.startswith("# "):
        story += [Spacer(1,18*mm),Paragraph(esc(line[2:]),s["TC"]),Paragraph("Architecture, implementation, operation, analysis, benchmarking, and future evolution.",s["SC"])]
    elif line.startswith("## "):
        story.append(Paragraph(esc(line[3:]),s["H1M"]))
    elif line.strip()=="":
        story.append(Spacer(1,2))
    elif line.startswith("    "):
        block=[]
        while i<len(lines) and (lines[i].startswith("    ") or lines[i].strip()==""):
            if lines[i].startswith("    "): block.append(lines[i][4:])
            i+=1
        story.append(Paragraph(esc("\\n".join(block)).replace("\\n","<br/>"),s["CM"]))
        continue
    elif line.startswith("- "):
        story.append(Paragraph("• "+esc(line[2:]),s["BLM"]))
    else:
        x=esc(line)
        x=re.sub(r"\\*\\*(.*?)\\*\\*",r"<b>\\1</b>",x)
        story.append(Paragraph(x,s["BM"]))
    i+=1
doc=SimpleDocTemplate(str(out),pagesize=A4,leftMargin=16*mm,rightMargin=16*mm,topMargin=15*mm,bottomMargin=15*mm,title="Go Data Profiler — Master Technical Copy")
doc.build(story,onFirstPage=footer,onLaterPages=footer)
