---
name: ste-writing
description: Write user-facing prose in Simplified Technical English (STE). Use when writing or editing documentation, READMEs, release notes, user-facing messages, commit messages, issues, or any non-code prose for this project.
---

# Simplified Technical English (STE)

Apply STE to documentation, READMEs, doc comments, user-facing messages,
release notes, issues, pull requests, and commit messages.

## Sentences

1. Keep sentences short. One idea per sentence.
2. Write in active voice. Name the actor.
3. Use simple present or simple past. Avoid future and perfect tenses when
   the present works.
4. Use the imperative for instructions: "Run the tests", not "You should run
   the tests" or "The tests should be run".

## Words

1. Use simple, everyday words. Prefer "use" over "utilize", "start" over
   "initiate", "stop" over "terminate".
2. One term per concept. Pick one word for a thing and use it everywhere.
   Never alternate between synonyms.
3. No idioms, metaphors, or slang. Write "the test fails", not "the test
   falls over".
4. Avoid vague quantifiers. Write the number: "three seconds", not "a short
   while".
5. Expand a new or domain abbreviation on first use. Self-evident standard
   abbreviations (URL, ID, HTTP, SQL) stand.

## Structure

1. Keep paragraphs short. One paragraph, one topic.
2. Use numbered lists for sequences. Use bullet lists for unordered facts.
3. State the goal before the steps.
4. State exceptions and warnings after the instruction they modify.

## What STE does not apply to

Code, code identifiers, sentinel names, log strings, and API names follow
the code conventions, not STE. Thinking and analysis are exempt.

## Why

STE reduces meaning-bandwidth on purpose. Short sentences with one idea are
easy to translate, easy to review, and hard to misread. This cost is
acceptable for prose and unacceptable for thinking, where full complexity is
needed.
