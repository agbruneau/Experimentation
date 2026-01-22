import { defineCollection, z } from 'astro:content';

const chapters = defineCollection({
  type: 'content',
  schema: z.object({
    title: z.string(),
    chapterNumber: z.number(),
    romanNumeral: z.string(),
    domain: z.enum([
      'introduction',
      'fondements',
      'applications',
      'donnees',
      'evenements',
      'standards',
      'resilience',
      'collaboration',
      'architecture',
      'case-study',
      'conclusion',
      'annexes'
    ]),
    domainColor: z.enum(['neutral', 'verb', 'noun', 'signal']).default('neutral'),
    description: z.string(),
  }),
});

export const collections = { chapters };
