# Handlat vision

Handlat makes everyday grocery shopping easier with simple shared lists and, over time, personal recipes that can become shopping lists.

## Purpose

Handlat began with a shopping list feature in ICA's app that worked well but lacked features its creator wanted, particularly support for personal recipes. Building Handlat makes it possible to shape those features directly.

This is a personal project for its creator, friends, and family. Its value comes from helping with everyday life and being enjoyable to build and maintain. There are no plans to charge for it, and growth or convincing people to switch from tools they already like is not a goal.

It is also a place to learn: to try libraries, explore ways of working, and gain experience with self-hosting. Experiments should remain compatible with a useful, manageable application.

## The everyday experience

The core experience is a grocery shopping list. A person can prepare a list, open it at the store, check off purchases, add forgotten items, and remove things they no longer need.

Lists can be shared. When people shop together, changes should appear in real time so everyone can see what has been added, removed, or checked off.

These everyday actions should feel straightforward on a phone.

## Recipes as a natural extension

Recipes should reduce the repeated work of turning meals into shopping lists.

The direction is to let people create and keep their own recipes, then add ingredients from those recipes to a shopping list. People should be able to use recipes they already have, including ones from websites and physical books.

Website imports and extracting recipes from photographs or scans are possible future ways to support this. They are ideas to explore, not commitments to particular features, technologies, or delivery dates.

Recipe management should build on the shopping experience without making basic list use cumbersome.

## Guiding principles

### Collect only what the application needs

Keep Handlat privacy friendly and avoid collecting unnecessary personal information. The current account approach uses a passkey and an automatically generated username, without requiring an email address or other personal details.

The application needs to store the content people use, such as lists and eventually recipes. Privacy means being deliberate about what is collected and why, rather than promising that no data exists.

### Make the web the primary home

Handlat is a web application, designed primarily for phone use and accessible on a computer as well. A larger screen may be particularly useful for creating and managing recipes.

The intended experience includes being installable on a phone's home screen as a progressive web app. Distribution should remain independent of app stores and should not require an Apple or Google account to access Handlat itself.

A native app is not currently a goal.

### Keep complexity proportional to the need

Choose solutions that are practical for a small, self-hosted personal project. Maintenance effort matters alongside the usefulness of a feature.

Installability does not by itself justify extensive caching, service workers, or a full offline architecture. Introduce those capabilities when a concrete need warrants their complexity.

### Let actual use guide development

Prioritize problems experienced by the creator, friends, and family. A feature does not need broad market appeal to be worthwhile, but it should have a clear purpose for the people using Handlat.

Preserve room for learning and experimentation while keeping the core shopping experience dependable.

## Direction and boundaries

The foundation is simple grocery lists, sharing, and real-time updates.

The next natural extension is personal recipe management and moving recipe ingredients into shopping lists. Importing recipes from websites or physical sources is a possible later enhancement.

Commercial growth, monetization, native app distribution, and elaborate offline infrastructure are not current objectives. Revisit these choices only when actual needs justify doing so.

This document describes direction, not a claim that every capability is already implemented or a fixed roadmap.

## Questions for future decisions

When proposing a change, contributors and agents should ask:

- Does this make everyday shopping easier for the people who use Handlat?
- Does it support shared lists or the path from personal recipes to shopping?
- Can the common actions remain clear and quick on a phone?
- Are we collecting only the information needed for the feature?
- Does it fit a web application that also works on a larger screen?
- Is its implementation and ongoing maintenance reasonable for a personal project?
- If it is an experiment, is the learning worth the complexity it introduces?

Use these questions to explain tradeoffs and keep changes aligned with the project's purpose.
