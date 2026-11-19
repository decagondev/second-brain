"""The lexical faithfulness scorer (sb-eval(5), Faithfulness). Strict and cheap: it
can't recognise a paraphrase, but it can't be talked into a high score either."""
import re

STOP = set("""
a about above after again against all also am an and any are as at be because been before being below between
both but by can could did do does doing down during each few for from further had has have having he her here
hers him his how into is it its itself just may might more most must need needs no nor not now of off on once
only or other our ours out over own same shall she should so some such than that the their theirs them then
there these they this those through to too under until up very was we were what when where which while who whom
why will with would you your yours
""".split())

WORD = re.compile(r"[a-z][a-z'-]{2,}")
NUMBER = re.compile(r"\d+(?:[.,]\d+)*")


def _numbers(text):
    return {n.replace(",", "") for n in NUMBER.findall(text)}


def content_words(text):
    return [w.strip("'-") for w in WORD.findall(text.lower()) if w.strip("'-") not in STOP and len(w.strip("'-")) >= 3]


def support(claim, cited_text):
    """Support of claim by cited_text, from 0 to 1."""
    nums = _numbers(claim)
    if nums - _numbers(cited_text):
        return 0.0  # a number the source doesn't say
    words = content_words(claim)
    if not words:
        return 1.0
    have = set(content_words(cited_text))
    return sum(w in have for w in words) / len(words)


def claim_support(c):
    """A claim's support: its own `support` if given (judge or mock), else lexical."""
    if "support" in c:
        return float(c["support"])
    return support(c.get("text", ""), c.get("cited_text", ""))
