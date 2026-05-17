import { describe, it, expect } from 'vitest'
import { useCursorPagination } from '../useCursorPagination'
describe('cursor import test', () => {
  it('can import the hook', () => {
    expect(typeof useCursorPagination).toBe('function')
  })
})
