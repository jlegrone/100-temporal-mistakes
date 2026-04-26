<draft here, do prework below first>
---

### Prework

*Cobbled together from the exercises in [Better Business Writing](https://www.udemy.com/course/betterbusinesswriting/) from Mark Morris.*

#### Background

Source material is the [100 Temporal Mistakes](https://github.com/jlegrone/100-temporal-mistakes) repo (currently 68 entries across 12 categories), modeled on Teivah's [100 Go Mistakes](https://github.com/teivah/100-go-mistakes). The talk is the conference-stage version of that material — a curated subset, organized around mental models rather than the table of contents.

#### Purposes

* Intended audience: Engineers who have started using Temporal — anywhere from "first week" to "few months in." Assumes basic familiarity with workflows and activities; doesn't re-teach first principles.
* Intended reviewers: Colleagues.
* Forum: Conference stage; recorded for YouTube.
* What do you want this document to communicate?
  * Temporal has real complications. Most of them exist for good reason though!
  * The audience can navigate those complications by recognizing the patterns ahead of time, so they leave with a mental toolkit, not a list of warnings.
* What outcomes do you want from publishing this?
  * Viewers go back to work and can confidently find and fix issues in their own Temporal workloads.
  * The repo becomes the post-talk reference for everything the talk didn't have time to cover.

#### Objective

SMART-ish:

* Specific: Deliver a 30–40 minute conference talk that exposes intermediate Temporal users to the most common mistakes and the design patterns that prevent them.
* Measurable: The audience asks questions and there is significant small group discussion afterward.
* Assignable: Just me; colleagues review.
* Realistic: Yes — the source material exists and only needs curation, narrative, and slides.
* Time-bound: 1 week

#### Reader

Target reader: An engineer who has been using Temporal for somewhere between a few weeks to a few years. Comfortable with the basics (workflows, activities, signals exist) but maybe hasn't yet been bitten by most of the mistakes in this talk.

Who else might read: More experienced Temporal users looking for a refresher or validation; engineers evaluating Temporal who want to understand the failure modes before adopting; people who land on the YouTube recording later via search.

What do we know about the target reader?
They are shipping real workloads under deadline. They may not have a dedicated Temporal platform team. They have read some Temporal docs but learn best from worked examples and direct experience. They came to a conference talk specifically because they want practical, applicable knowledge — not a vendor pitch and not a tutorial.

Why are we writing to them? Is writing the best way to communicate?
A talk is the right medium because the *patterns* matter more than the per-mistake detail — a talk forces curation and narrative. The repo is there for anyone who wants the full reference afterward.

What do they want?
To stop being surprised by Temporal in production. To recognize bad patterns in their own code. To leave with a small number of habits they can apply in their own work.

What will readers' feelings be after reading this? How does that affect what we write?
Goal: empowered, not overwhelmed. They should walk away thinking *"Temporal is complicated for good reasons, and I now know what to look out for"* — not *"Temporal is a minefield."*

Pitfalls to avoid:
- Listing too many mistakes flatly — reads as a minefield.
- Sounding judgmental (the "mistakes" framing of the title already leans this way; avoid judgement everywhere else).
- Giving the impression that using Temporal correctly is hopeless or requires constant vigilance.

#### Voice

Speaking as myself, but also representing collective experience of engineers using Temporal at Datadog. An experienced and knowledgeable end user of Temporal sharing what I've learned the hard way.

How to come across:
- Pragmatic, hands-on, grounded in real experience.
- Confident but not preachy. Each "mistake" should land as *"here's a thing that surprised me / a team I worked with"* — not *"here's what you're doing wrong."*
- The talk closes with a small set of recommendations and design patterns that prevent most of the mistakes shown — leaving the audience with a manageable toolkit, not a checklist of fears.

#### Potential → Outline

(working section: "idea storm" notes start in Potential, migrate towards Outline; be ruthless\!)

##### Potential

* \<start by sketching out everything you *could* include\>

##### Outline

* \<move things from "potential", but *leave some behind\!*\>
