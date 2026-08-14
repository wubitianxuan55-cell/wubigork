import { describe, expect, it } from 'vitest'
import { downloadFileName, mediaExtension, mediaIsVideo } from './media'

describe('mediaExtension', () => {
  it('names t2v webp output as .webp instead of .mp4', () => {
    expect(mediaExtension('data:image/webp;base64,AAAA')).toBe('.webp')
    expect(mediaExtension('data:video/webp;base64,AAAA')).toBe('.webp')
  })

  it('names real videos by container', () => {
    expect(mediaExtension('data:video/mp4;base64,AAAA')).toBe('.mp4')
    expect(mediaExtension('data:video/webm;base64,AAAA')).toBe('.webm')
    expect(mediaExtension('data:video/quicktime;base64,AAAA')).toBe('.mov')
  })

  it('names images and defaults to png', () => {
    expect(mediaExtension('data:image/gif;base64,AAAA')).toBe('.gif')
    expect(mediaExtension('data:image/jpeg;base64,AAAA')).toBe('.jpg')
    expect(mediaExtension('data:image/png;base64,AAAA')).toBe('.png')
    expect(mediaExtension('')).toBe('.png')
    expect(mediaExtension('not-a-data-url')).toBe('.png')
  })
})

describe('downloadFileName', () => {
  it('uses .webp for t2v webp output instead of fixed .mp4', () => {
    expect(downloadFileName({ image: 'data:image/webp;base64,AAAA', seed: 42 }, 123456))
      .toBe('gaea-123456-seed42.webp')
  })

  it('uses .mp4 for real video data', () => {
    expect(downloadFileName({ image: 'data:video/mp4;base64,AAAA', seed: 7 }, 1))
      .toBe('gaea-1-seed7.mp4')
  })

  it('defaults to .png for plain images and empty data', () => {
    expect(downloadFileName({ image: 'data:image/png;base64,AAAA', seed: 1 }, 2))
      .toBe('gaea-2-seed1.png')
    expect(downloadFileName({ image: '', seed: 9 }, 3)).toBe('gaea-3-seed9.png')
  })
})

describe('mediaIsVideo', () => {
  it('detects video data URLs only', () => {
    expect(mediaIsVideo('data:video/mp4;base64,AAAA')).toBe(true)
    expect(mediaIsVideo('data:video/webm;base64,AAAA')).toBe(true)
    expect(mediaIsVideo('data:image/webp;base64,AAAA')).toBe(false)
    expect(mediaIsVideo('data:image/png;base64,AAAA')).toBe(false)
    expect(mediaIsVideo('')).toBe(false)
  })
})
