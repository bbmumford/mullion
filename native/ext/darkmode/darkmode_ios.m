//go:build ios

#import <UIKit/UIKit.h>
#import "darkmode_ios.h"

extern void mullionExtDarkmodeIOSUpdate(int isDark);

@interface MullionDarkModeObserver : UIView
- (void)syncAppearance;
@end

@implementation MullionDarkModeObserver

- (instancetype)init {
    self = [super initWithFrame:CGRectZero];
    if (self) {
        self.userInteractionEnabled = NO;
        if (@available(iOS 17.0, *)) {
            [self registerForTraitChanges:@[[UITraitUserInterfaceStyle class]]
                               withAction:@selector(syncAppearance)];
        }
    }
    return self;
}

- (void)didMoveToWindow {
    [super didMoveToWindow];
    [self syncAppearance];
}

- (void)traitCollectionDidChange:(UITraitCollection *)previousTraitCollection {
    [super traitCollectionDidChange:previousTraitCollection];
    if (@available(iOS 17.0, *)) {
        return;
    }
    if (self.traitCollection.userInterfaceStyle != previousTraitCollection.userInterfaceStyle) {
        [self syncAppearance];
    }
}

- (void)syncAppearance {
    mullionExtDarkmodeIOSUpdate(self.traitCollection.userInterfaceStyle == UIUserInterfaceStyleDark);
}

@end

int mullion_ext_darkmode_ios_start_observer(void *rootViewController) {
    @autoreleasepool {
        UIViewController *root = (__bridge UIViewController *)rootViewController;
        if (root == nil) {
            return [UITraitCollection currentTraitCollection].userInterfaceStyle == UIUserInterfaceStyleDark;
        }
        MullionDarkModeObserver *observer = [[MullionDarkModeObserver alloc] init];
        [root.view addSubview:observer];
        return observer.traitCollection.userInterfaceStyle == UIUserInterfaceStyleDark;
    }
}
